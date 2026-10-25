package usecase

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/enum"
	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func CreateDockerfile(config models.Cluster) error {

	chanErrors := make(chan error, len(config.Datanodes)+1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		dockerComposeFileMaster := createMasterCompose(config)

		if err := os.WriteFile("new_cluster/master/docker-compose.master.yml", []byte(dockerComposeFileMaster), 0644); err != nil {
			chanErrors <- err
			return
		}
	}()

	privateKeyPath, err := getPathPrivateKey()
	if err != nil {
		return fmt.Errorf("não foi possível localizar a chave SSH: %w", err)
	}

	const createWorkerComposeCommand = `mkdir -p "$HOME/S.H.A.N.K.S/new_cluster/worker" && cat > "$HOME/S.H.A.N.K.S/new_cluster/worker/docker-compose.worker.yml"`

	for _, datanode := range config.Datanodes {
		if datanode == nil {
			return fmt.Errorf("datanode inválido na configuração")
		}

		wg.Add(1)
		go func(datanode *models.Datanode) {
			defer wg.Done()

			baseImage, _ := mlImageConfig(datanode.GPU)
			var dockerComposeFile string
			if datanode.GPU {
				dockerComposeFile = fmt.Sprintf(`
services:
  worker:
    build:
      context: ..
      dockerfile: worker/Dockerfile.worker
      args:
        SHANKS_BASE_IMAGE: %s
    container_name: &DATANODE_NAME %s
    env_file: ../../.env
    network_mode: "host"
    hostname: *DATANODE_NAME
    environment:
      NVIDIA_DRIVER_CAPABILITIES: compute,utility
      SPARK_WORKER_MEMORY: "%s"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    volumes:
      - datanode_data:/home/hadoop/hadoop/hdfs/datanode

  node-exporter-worker:
    image: prom/node-exporter
    network_mode: service:worker


volumes:
  datanode_data:
`, baseImage, datanode.Name, datanode.SparkLimit)

			} else {
				dockerComposeFile = fmt.Sprintf(`
services:
  worker:
    build:
      context: ..
      dockerfile: worker/Dockerfile.worker
      args:
        SHANKS_BASE_IMAGE: %s
    container_name: &DATANODE_NAME %s
    env_file: ../../.env
    network_mode: "host"
    hostname: *DATANODE_NAME
    environment:
      SPARK_WORKER_MEMORY: "%s"
    volumes:
      - datanode_data:/home/hadoop/hadoop/hdfs/datanode

  node-exporter-worker:
    image: prom/node-exporter
    network_mode: service:worker


volumes:
  datanode_data:
`, baseImage, datanode.Name, datanode.SparkLimit)
			}

			if _, err := runSSHCommandBuffer(
				datanode.IP,
				"22",
				datanode.User,
				privateKeyPath,
				createWorkerComposeCommand,
				dockerComposeFile,
			); err != nil {
				chanErrors <- fmt.Errorf("não foi possível criar o docker-compose.worker.yml no datanode %s: %w", datanode.Name, err)
			}
		}(datanode)

	}

	wg.Wait()
	close(chanErrors)

	for err := range chanErrors {
		if err != nil {
			return fmt.Errorf("erro ao criar arquivos docker-compose: %w", err)
		}
	}

	return nil
}

func createMasterCompose(config models.Cluster) string {
	baseImage, _ := mlImageConfig(config.Namenode.GPU)
	masterGPUConfig := ""
	if config.Namenode.GPU {
		masterGPUConfig = `    environment:
      NVIDIA_DRIVER_CAPABILITIES: compute,utility
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
`
	}

	var tableFormat string

	if config.Lakehouse.Enabled {

		if strings.ToLower(config.Lakehouse.TableFormat) == enum.Iceberg.String() {
			tableFormat = `  spark-iceberg:
    image: tabulario/spark-iceberg:3.5.5_1.8.1
    container_name: spark-iceberg
    network_mode: host
    depends_on:
      - jupyter
    volumes:
      - ./warehouse:/home/iceberg/warehouse
      - ./notebooks:/home/iceberg/notebooks/notebooks
    ports:
      - 8889:8888
      - 9050:8080
      - 10000:10000
      - 10001:10001`
		}
	}

	return fmt.Sprintf(`
services:
  master:
    build:
      context: ..
      dockerfile: master/Dockerfile.master
      args:
        SHANKS_BASE_IMAGE: %s
    hostname: master
    container_name: hadoop-master
    network_mode: host
    env_file: ../../.env
%s    volumes:
      - namenode_data:/home/hadoop/hadoop/hdfs/namenode
      - namenode_data_datanode:/home/hadoop/hadoop/hdfs/datanode
      - hadoop_conf:/home/hadoop/hadoop/etc/hadoop

  jupyter:
    image: shanks-jupyter:ml311
    build:
      context: ..
      dockerfile: jupyter/Dockerfile.jupyter
    container_name: shanks-jupyter
    network_mode: host
    env_file: ../../.env
    environment:
      HADOOP_CONF_DIR: /etc/hadoop/conf
      YARN_CONF_DIR: /etc/hadoop/conf
      SPARK_CONF_DIR: /home/jovyan/spark-conf
      PYSPARK_PYTHON: /opt/shanks-venv/bin/python
      PYSPARK_DRIVER_PYTHON: /opt/conda/envs/shanks311/bin/python
      SPARK_LOCAL_IP: %s
    volumes:
      - hadoop_conf:/etc/hadoop/conf:ro
      - notebooks:/home/jovyan/work
      - /etc/hosts:/etc/hosts:ro
      - ../jupyter/start-jupyter.sh:/opt/shanks/start-jupyter.sh:ro
    command: ["bash", "/opt/shanks/start-jupyter.sh"]
    depends_on:
      - master

  prometheus:
    image: prom/prometheus
    container_name: prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    network_mode: host

  grafana:
    image: grafana/grafana
    container_name: grafana
    volumes:
      - grafana:/var/lib/grafana
      - ../grafana/provisioning:/etc/grafana/provisioning
      - ../grafana/files:/var/lib/grafana/files
      - ../grafana/datasources:/etc/grafana/provisioning/datasources
    depends_on:
      - prometheus
    network_mode: host

  node-exporter-master:
    image: prom/node-exporter
    container_name: exporter-master
    network_mode: service:master

%s

volumes:
  grafana:
  namenode_data:
  namenode_data_datanode:
  hadoop_conf:
  notebooks:`, baseImage, masterGPUConfig, config.Namenode.IP, tableFormat)
}

func mlImageConfig(gpu bool) (image, torchFlavor string) {
	if gpu {
		return "hadoop-base:ml311-cu126", "cu126"
	}

	return "hadoop-base:ml311-cpu", "cpu"
}
