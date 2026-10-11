package usecase

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

/// ENVIAR SSH KEY DA MASTER PARA OS NODES QUE ESTÃO NO ARQUIVO CONFIG.YML
/// ENVIAR DE FORMA ASSINCRONA

func ConfigHosts(c models.Config) error {

	var wg sync.WaitGroup

	nameNode := fmt.Sprintf("%s %s\n", c.Cluster.Namenode.IP, c.Cluster.Namenode.Name)

	var bufferDatanodes strings.Builder

	bufferDatanodes.WriteString(nameNode)
	bufferDatanodes.WriteString("\n")

	// POPULAR O BUFFER COM OS DADOS DE CADA DATANODE
	for _, datanode := range c.Cluster.Datanodes {

		dataNodeString := fmt.Sprintf("%s %s", datanode.IP, datanode.Name)

		bufferDatanodes.WriteString(dataNodeString)
		bufferDatanodes.WriteString("\n")

	}

	privateKey, err := getPathPrivateKey()
	if err != nil {
		return err
	}

	chanErrorsDatanode := make(chan models.Datanode, len(c.Cluster.Datanodes)+1)

	// POPULAR O ARQUIVO /ETC/HOSTS DE CADA DATANODE COM O BUFFER DE DADOS
	for _, datanode := range c.Cluster.Datanodes {

		wg.Add(1)
		go func(datanode *models.Datanode) {
			defer wg.Done()
			_, err := runSSHCommandBuffer(datanode.IP, "22", datanode.User, privateKey, "sudo -n /usr/local/bin/config-hosts-shanks.sh", bufferDatanodes.String())
			if err != nil {
				chanErrorsDatanode <- *datanode
			}

		}(datanode)
	}

	wg.Wait()
	close(chanErrorsDatanode)

	for errDatanode := range chanErrorsDatanode {
		return fmt.Errorf("erro ao configurar /etc/hosts no datanode %s (%s)", errDatanode.Name, errDatanode.IP)
	}

	command := exec.Command("sudo", "-n", "/usr/local/bin/config-hosts-shanks.sh")

	command.Stdin = strings.NewReader(bufferDatanodes.String())

	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"erro ao executar config-hosts-shanks: %w\nsaída: %s",
			err,
			output,
		)
	}

	return nil
}

// func generateCommandToUpdateHosts(password string, bufferDatanodes strings.Builder) string {
// 	return fmt.Sprintf(`echo "%s" | sudo -S sh -c " { sed '/# START S.H.A.N.K.S #/,/# END S.H.A.N.K.S #/d' /etc/hosts; echo  '# START S.H.A.N.K.S #'; echo '%s'; echo '# END S.H.A.N.K.S #'; } > /etc/hosts.tmp && mv /etc/hosts.tmp /etc/hosts "`, password, bufferDatanodes.String())
// }

// func pruneConfigHosts(password string) string {
// 	return fmt.Sprintf(`echo "%s" | sudo -S sh -c "sed -i '/# START S.H.A.N.K.S #/,/# END S.H.A.N.K.S #/d' /etc/hosts"`, password)
// }
