package usecase

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func StopCluster(configCluster models.Config) error {

	var wg sync.WaitGroup

	chErrors := make(chan error, len(configCluster.Cluster.Datanodes)+1)

	time.Sleep(time.Second * 3)

	home, err := getHomeDir()

	if err != nil {
		return err
	}

	commandMaster := fmt.Sprintf("cd %s/S.H.A.N.K.S/new_cluster/master && docker compose -f 'docker-compose.master.yml' down", home)
	commandWorkers := "cd $HOME/S.H.A.N.K.S/new_cluster/worker && docker compose -f 'docker-compose.worker.yml' down"

	privatePathSSHKey, err := getPathPrivateKey()
	if err != nil {
		return err
	}

	logInfo("Parando container da master...")

	wg.Add(1)
	go func(chErrors chan error) {
		defer wg.Done()
		out, err := exec.Command("bash", "-c", commandMaster).CombinedOutput()
		if err != nil {
			logFailure("Erro ao parar container da master")
			chErrors <- fmt.Errorf("erro ao subir a master: %v\nLogs: %s", err, string(out))
			chErrors <- err
			return
		}
		logSuccess("Container master parado com sucesso")
	}(chErrors)

	for _, datanode := range configCluster.Cluster.Datanodes {
		wg.Add(1)
		go func(chErrors chan error) {
			defer wg.Done()
			logInfo("Parando container do datanode %s", datanode.Name)
			_, err = runSSHCommand(datanode.IP, "22", datanode.User, privatePathSSHKey, commandWorkers)
			if err != nil {
				logFailure("Erro ao parar container do datanode %s", datanode.Name)
				chErrors <- fmt.Errorf("erro ao subir container no datanode %s: %v", datanode.Name, err)
				return
			}
			logSuccess("Container do datanode %s parado com sucesso", datanode.Name)
		}(chErrors)

	}

	wg.Wait()

	close(chErrors)

	for err := range chErrors {
		if err != nil {
			return err
		}
	}
	logSuccess("☠️🗡️ Cluster parado com sucesso")
	return nil
}
