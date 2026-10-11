package usecase

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func PruneCluster(configCluster *models.Config) error {

	time.Sleep(time.Second * 3)
	logInfo("Destruindo todo o cluster")

	var wg sync.WaitGroup

	home, err := getHomeDir()
	if err != nil {
		logFailure("Erro ao obter diretório home")
		return err
	}

	commandDownContainersMaster := fmt.Sprintf("cd %s/S.H.A.N.K.S/new_cluster/master && docker compose -f docker-compose.master.yml down -v", home)
	commandRemoveImagesMaster := "docker images -q master-master:latest | xargs -r docker rmi -f &&  docker images -q hadoop-base:latest | xargs -r docker rmi -f; docker images -q grafana/grafana:latest | xargs -r docker rmi -f; docker images -q prom/node-exporter:latest | xargs -r docker rmi -f; docker images -q prom/prometheus:latest | xargs -r docker rmi -f"

	commandDownContainersWorker := "cd $HOME/S.H.A.N.K.S/new_cluster/worker && docker compose -f docker-compose.worker.yml down -v"
	commandRemoveImagesWorker := "docker images -q worker-worker:latest | xargs -r docker rmi -f && docker images -q hadoop-base:latest | xargs -r docker rmi -f; docker images -q prom/node-exporter:latest | xargs -r docker rmi -f"

	logInfo("Desfazendo configuarções do arquivo /etc/hosts feita pelo S.H.A.N.K.S")
	if err := exec.Command("bash", "-c", "/usr/local/bin/remove-config-hosts.sh").Run(); err != nil {
		logFailure("Erro ao desfazer configurações do arquivo /etc/hosts")
		return err
	}
	logSuccess("Configuração do /etc/hosts desfeita com sucesso")

	logInfo("Derrubando containers do master")
	if err := exec.Command("bash", "-c", commandDownContainersMaster).Run(); err != nil {
		logFailure("Erro ao derrubar containers do master")
		return err
	}
	logSuccess("Containers do master derrubados")

	if err := exec.Command("bash", "-c", commandRemoveImagesMaster).Run(); err != nil {
		fmt.Println(err)
		logFailure("Erro ao remover imagens do master")
		return err
	}
	logSuccess("Imagens do master removidas")

	pathPrivateKey, err := getPathPrivateKey()
	if err != nil {
		logFailure("Erro ao obter chave privada")
		return err
	}

	chanErrorsDatanode := make(chan error, len(configCluster.Cluster.Datanodes)+1)

	for _, datanode := range configCluster.Cluster.Datanodes {
		wg.Add(1)
		go func(datanode *models.Datanode) {
			defer wg.Done()
			logInfo("Removendo tag do S.H.A.N.K.S do arquivo /etc/hosts do datanode %s", datanode.Name)

			if _, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, "/usr/local/bin/remove-config-hosts.sh"); err != nil {
				logFailure("Erro ao remover tag do S.H.A.N.K.S do arquivo /etc/hosts do datanode %s", datanode.Name)
				chanErrorsDatanode <- err
				return
			}
			logInfo("Tag do S.H.A.N.K.S removida do arquivo /etc/hosts do datanode %s", datanode.Name)

			logInfo("Derrubando containers do datanode %s", datanode.Name)
			if _, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandDownContainersWorker); err != nil {
				logFailure("Erro ao derrubar containers do datanode %s", datanode.Name)
				chanErrorsDatanode <- err
				return
			}
			logInfo("Containers do datanode %s derrubados", datanode.Name)

			if _, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandRemoveImagesWorker); err != nil {
				logFailure("Erro ao remover imagens do datanode %s", datanode.Name)
				chanErrorsDatanode <- err
				return
			}
			logInfo("Imagens do datanode %s removidas", datanode.Name)
		}(datanode)

	}

	wg.Wait()
	close(chanErrorsDatanode)

	for err := range chanErrorsDatanode {
		if err != nil {
			return fmt.Errorf("Erro ao remover configuração do datanode: %v", err)
		}
	}

	return nil

}
