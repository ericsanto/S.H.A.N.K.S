package usecase

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
	"github.com/ericsanto/S.H.A.N.K.S/tui"
)

func Health(c models.Config) (*tui.Model, error) {

	var wg sync.WaitGroup

	// pathPrivateKey, err := getPathPrivateKey()
	// if err != nil {
	// 	logFailure("Erro ao obter chave privada")
	// 	return nil, err
	// }

	chStats := make(chan tui.NodeHealth, len(c.Cluster.Datanodes)+1) // +1 para o master
	chModel := make(chan tui.Model, 1)

	wg.Add(1)
	go func(namenode models.Namenode, wg *sync.WaitGroup, chStats chan<- tui.NodeHealth) {

		defer wg.Done()
		commandHdfs := fmt.Sprintf("docker exec %s hdfs dfsadmin -report", namenode.Name)
		// commandYarn := fmt.Sprintf("docker exec %s yarn node -list", namenode.Name)
		commandDockerStats := fmt.Sprintf("docker stats --no-stream --format  \"CPU: {{.CPUPerc}}\" %s; docker stats --no-stream --format  \"Memory: {{.MemUsage}}\" %s; docker inspect --format \"status: {{.State.Status}}\" %s", namenode.Name, namenode.Name, namenode.Name)

		dockerStatus, err := exec.Command("bash", "-c", commandDockerStats).Output()
		if err != nil {
			logFailure("Erro ao verificar status do Docker no master %s", namenode.Name)
			chStats <- tui.NodeHealth{Name: namenode.Name, DockerStatus: "Erro"}
			return
		}

		nodeH := tui.NodeHealth{Name: namenode.Name}
		nodeH.GetDockerData(string(dockerStatus))

		hdfsStatus, err := exec.Command("bash", "-c", commandHdfs).Output()
		if err != nil {
			logFailure("Erro ao verificar status do HDFS no master %s", namenode.Name)
			chStats <- tui.NodeHealth{Name: namenode.Name}
			return
		}

		model := tui.Model{}
		model.GetHDFSData(string(hdfsStatus))
		nodeH.GetHDFSDataNode(string(hdfsStatus), namenode.Name)

		// yarnStatus, err := exec.Command("bash", "-c", commandYarn).Output()
		// if err != nil {
		// 	logFailure("Erro ao verificar status do YARN no master %s", namenode.Name)
		// 	chStats <- tui.NodeHealth{Name: namenode.Name, YARNStatus: "Erro"}
		// 	return
		// }

		// nodeH.GetHDFSData(string(hdfsStatus), namenode.Name)

		chModel <- model
		chStats <- nodeH

	}(c.Cluster.Namenode, &wg, chStats)

	pathPrivateKey, err := getPathPrivateKey()
	if err != nil {
		logFailure("Erro ao obter chave privada")
		return nil, err
	}

	for _, datanode := range c.Cluster.Datanodes {

		commandHdfs := fmt.Sprintf("docker exec %s hdfs dfsadmin -report", datanode.Name)
		// commandYarn := fmt.Sprintf("docker exec  %s yarn yarn node -list", datanode.Name)
		commandDockerStats := fmt.Sprintf("docker stats --no-stream --format  \"CPU: {{.CPUPerc}}\" %s; docker stats --no-stream --format  \"Memory: {{.MemUsage}}\" %s; docker inspect --format \"status: {{.State.Status}}\" %s", datanode.Name, datanode.Name, datanode.Name)

		wg.Add(1)
		go func(datanode models.Datanode, wg *sync.WaitGroup, chStats chan<- tui.NodeHealth) {
			defer wg.Done()
			dockerStatus, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandDockerStats)

			if err != nil {
				logFailure("Erro ao verificar status do Docker no datanode %s", datanode.Name)
				chStats <- tui.NodeHealth{Name: datanode.Name, DockerStatus: "Erro"}
				return
			}

			hdfsStatus, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandHdfs)
			if err != nil {
				logFailure("Erro ao verificar status do HDFS no datanode %s", datanode.Name)
				chStats <- tui.NodeHealth{Name: datanode.Name}
				return
			}

			// yarnStatus, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, commandYarn)
			// if err != nil {
			// 	logFailure("Erro ao verificar status do YARN no datanode %s", datanode.Name)
			// 	chStats <- tui.NodeHealth{Name: datanode.Name, YARNStatus: "Erro"}
			// 	return
			// }

			nodeH := tui.NodeHealth{Name: datanode.Name}
			nodeH.GetDockerData(string(dockerStatus))
			nodeH.GetHDFSDataNode(string(hdfsStatus), datanode.Name)

			chStats <- nodeH
		}(*datanode, &wg, chStats)
	}

	wg.Wait()
	close(chModel)
	close(chStats)

	return populateModelHealth(chStats, chModel), nil

}

func populateModelHealth(ch <-chan tui.NodeHealth, chModel <-chan tui.Model) *tui.Model {

	model := <-chModel

	for nodeHealth := range ch {
		model.NodesHealth = append(model.NodesHealth, nodeHealth)
	}

	return &model

}
