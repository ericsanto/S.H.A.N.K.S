package usecase

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func PruneCluster(configCluster *models.Config) error {

	time.Sleep(time.Second * 3)
	logInfo("Destruindo todo o cluster")

	home, err := getHomeDir()
	if err != nil {
		logFailure("Erro ao obter diretório home")
		return err
	}

	commandRemoveConfigEtcHost := "sed -i '/# START S.H.A.N.K.S #/,/# END S.H.A.N.K.S #/d' /etc/hosts"
	commandDownContainersMaster := fmt.Sprintf("cd %s/S.H.A.N.K.S/new_cluster/master && docker compose -f docker-compose.master.yml down -v", home)
	commandRemoveImagesMaster := "docker images -q master-master:latest | xargs -r docker rmi -f &&  docker images -q hadoop-base:latest | xargs -r docker rmi -f; docker images -q grafana/grafana:latest | xargs -r docker rmi -f; docker images -q prom/node-exporter:latest | xargs -r docker rmi -f; docker images -q prom/prometheus:latest | xargs -r docker rmi -f"

	commandDownContainersWorker := "cd $HOME/S.H.A.N.K.S/new_cluster/worker && docker compose -f docker-compose.worker.yml down -v"
	commandRemoveImagesWorker := "docker images -q worker-worker:latest | xargs -r docker rmi -f && docker images -q hadoop-base:latest | xargs -r docker rmi -f; docker images -q prom/node-exporter:latest | xargs -r docker rmi -f"

	logInfo("Desfazendo configuarções do arquivo /etc/hosts feita pelo S.H.A.N.K.S")
	if err := exec.Command("bash", "-c", commandRemoveConfigEtcHost).Run(); err != nil {
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

	for _, datanode := range configCluster.Cluster.Datanodes {
		// logInfo("Desfazendo configuração do arquivo /etc/hosts feita pelo S.H.A.N.K.S no datanode %s", datanode.Name)
		// if _, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandRemoveConfigEtcHost); err != nil {
		// 	logFailure("Erro ao desfazer configuração do arquivo /etc/hosts no datanode %s", datanode.Name)
		// 	return err
		// }
		// logInfo("Configuração do /etc/hosts desfeita com sucesso no datanode %s", datanode.Name)

		logInfo("Removendo tag do S.H.A.N.K.S do arquivo /etc/hosts do datanode %s", datanode.Name)

		var pass string

		for {

			pass = readPasswordInteractive(datanode.User, datanode.IP)

			if pass == "" {
				logFailure("Senha não pode ser vazia ou inválida")
				continue
			}

			break
		}

		if _, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, pruneConfigHosts(pass)); err != nil {
			logFailure("Erro ao remover tag do S.H.A.N.K.S do arquivo /etc/hosts do datanode %s", datanode.Name)
			return err
		}
		logInfo("Tag do S.H.A.N.K.S removida do arquivo /etc/hosts do datanode %s", datanode.Name)

		logInfo("Derrubando containers do datanode %s", datanode.Name)
		if _, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandDownContainersWorker); err != nil {
			logFailure("Erro ao derrubar containers do datanode %s", datanode.Name)
			return err
		}
		logInfo("Containers do datanode %s derrubados", datanode.Name)

		if _, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandRemoveImagesWorker); err != nil {
			logFailure("Erro ao remover imagens do datanode %s", datanode.Name)
			return err
		}
		logInfo("Imagens do datanode %s removidas", datanode.Name)
	}

	return nil

}
