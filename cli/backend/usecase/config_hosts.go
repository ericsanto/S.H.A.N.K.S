package usecase

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

/// ENVIAR SSH KEY DA MASTER PARA OS NODES QUE ESTÃO NO ARQUIVO CONFIG.YML
/// ENVIAR DE FORMA ASSINCRONA

func ConfigHosts(c models.Config) error {

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

	// POPULAR O ARQUIVO /ETC/HOSTS DE CADA DATANODE COM O BUFFER DE DADOS
	for _, datanode := range c.Cluster.Datanodes {
		// passwordDatanode := readPasswordInteractive(datanode.User, datanode.IP)

		// if passwordDatanode == "" {
		// 	return fmt.Errorf("senha nao pode ser vazia")
		// }

		// command := generateCommandToUpdateHosts(passwordDatanode, bufferDatanodes)

		// EXECUTA O COMANDO PARA ATUALIZAR O ARQUIVO /ETC/HOSTS DO DATANODE VIA SSH
		_, err := runSSHCommandBuffer(datanode.IP, "22", datanode.User, privateKey, "sudo -n /usr/local/bin/config-hosts-shanks.sh", bufferDatanodes.String())
		if err != nil {
			return err
		}

	}

	// passwordNamenode := readPasswordInteractive(c.Cluster.Namenode.User, c.Cluster.Namenode.IP)

	// command := generateCommandToUpdateHosts(passwordNamenode, bufferDatanodes)

	// POPULA O ARQUIVO /ETC/HOSTS DO NAMENODE COM O BUFFER DE DADOS

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

func pruneConfigHosts(password string) string {
	return fmt.Sprintf(`echo "%s" | sudo -S sh -c "sed -i '/# START S.H.A.N.K.S #/,/# END S.H.A.N.K.S #/d' /etc/hosts"`, password)
}
