package usecase

import (
	"fmt"
	"os"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func CreateDockerfile(config models.Cluster) error {

	var dockerComposeFile string

	dockerComposeFileMaster := fmt.Sprintf(`
services:
  master:
    build:
      context: ..
      dockerfile: master/Dockerfile.master
    hostname: master
    container_name: hadoop-master
    network_mode: host
    env_file: ../../.env
    volumes:
      - namenode_data:/home/hadoop/hadoop/hdfs/namenode
      - namenode_data_datanode:/home/hadoop/hadoop/hdfs/datanode
      - hadoop_conf:/home/hadoop/hadoop/etc/hadoop

  jupyter:
    image: quay.io/jupyter/pyspark-notebook:x86_64-spark-3.5.3
    container_name: shanks-jupyter
    network_mode: host
    env_file: ../../.env
    environment:
      HADOOP_CONF_DIR: /etc/hadoop/conf
      YARN_CONF_DIR: /etc/hadoop/conf
      SPARK_CONF_DIR: /home/jovyan/spark-conf
      PYSPARK_PYTHON: python3
      PYSPARK_DRIVER_PYTHON: python3
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

volumes:
  grafana:
  namenode_data:
  namenode_data_datanode:
  hadoop_conf:
  notebooks:`, config.Namenode.IP)

	if err := os.WriteFile("new_cluster/master/docker-compose.master.yml", []byte(dockerComposeFileMaster), 0644); err != nil {
		return err
	}

	for _, datanode := range config.Datanodes {

		if datanode.GPU {
			dockerComposeFile = fmt.Sprintf(`
services:
  worker:
    build: 
      context: ..
      dockerfile: worker/Dockerfile.worker
    container_name: &DATANODE_NAME %s
    env_file: ../../.env
    network_mode: "host"
    hostname: *DATANODE_NAME
    environment:
      NVIDIA_DRIVER_CAPABILITIES: compute,utility
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
`, datanode.Name)

		} else {
			dockerComposeFile = fmt.Sprintf(`
services:
  worker:
    build: 
      context: ..
      dockerfile: worker/Dockerfile.worker
    container_name: &DATANODE_NAME %s
    env_file: ../../.env
    network_mode: "host"
    hostname: *DATANODE_NAME
    volumes:
      - datanode_data:/home/hadoop/hadoop/hdfs/datanode
  node-exporter-worker:
    image: prom/node-exporter
    network_mode: service:worker


volumes:
  datanode_data:
`, config.Namenode.Name)
		}
	}

	if err := os.WriteFile("new_cluster/worker/docker-compose.worker.bak.yml", []byte(dockerComposeFile), 0644); err != nil {
		return err
	}
	return nil
}
