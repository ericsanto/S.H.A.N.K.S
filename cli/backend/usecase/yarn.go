package usecase

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func ConfigureYarnLimits(configCluster models.Config) error {

	yarnMaster, err := configCluster.Cluster.Namenode.ConvertToYarnLimit()

	if err != nil {
		return fmt.Errorf("erro ao converter o limite de memória do Yarn do Namenode: %v", err)
	}

	if err := configCluster.Cluster.Namenode.VerifiyValueMinYarnRequirements(); err != nil {
		return err
	}

	pathPrivateKey, err := getPathPrivateKey()
	if err != nil {
		return err
	}

	validatedYarnMaster, err := validateYarnLimits(configCluster.Cluster.Namenode.IP, configCluster.Cluster.Namenode.User, pathPrivateKey, yarnMaster, "namenode")
	if err != nil {
		return err
	}

	configCluster.Cluster.Namenode.YarnLimit = *validatedYarnMaster

	yarnWorkers := make([]models.Datanode, len(configCluster.Cluster.Datanodes)-1)

	for _, datanode := range configCluster.Cluster.Datanodes {

		yarnLimit, err := datanode.ConvertToYarnLimit()
		if err != nil {
			return fmt.Errorf("erro ao converter o limite de memória do Yarn do Datanode %s: %v", datanode.Name, err)
		}

		if err := datanode.VerifiyValueMinYarnRequirements(); err != nil {
			return err
		}

		validatedYarnLimit, err := validateYarnLimits(datanode.IP, datanode.User, pathPrivateKey, yarnLimit, "datanode")
		if err != nil {
			return err
		}

		datanode.YarnLimit = *validatedYarnLimit
		yarnWorkers = append(yarnWorkers, *datanode)

		fmt.Println(yarnWorkers)
	}

	if err := insertYarnLimitsToEnvFile(configCluster.Cluster.Namenode, yarnWorkers); err != nil {
		return err
	}

	return nil

}

func validateYarnLimits(ip, user, pathPrivateKey string, yarnLimit float64, config string) (*string, error) {

	if config == "namenode" {

		out, err := exec.Command("bash", "-c", "free -m | awk 'NR==2{print $2}'").Output()

		if err != nil {
			return nil, err
		}

		result := string(out)

		memory := strings.TrimSpace(result)

		intMemory, err := strconv.ParseFloat(strings.TrimSpace(memory), 64)
		if err != nil {
			return nil, err
		}

		if intMemory < yarnLimit {
			yarnLimit = intMemory * 70 / 100
			yarnLimitStr := strconv.FormatFloat(yarnLimit, 'f', -1, 64)
			return &yarnLimitStr, nil

		} else {
			yarnLimitStr := strconv.FormatFloat(yarnLimit, 'f', -1, 64)
			return &yarnLimitStr, nil
		}

	} else {
		memory, err := runSSHCommand(ip, "22", user, pathPrivateKey, "free -m | awk 'NR==2{print $2}'")
		if err != nil {
			return nil, err
		}

		intMemory, err := strconv.ParseFloat(strings.TrimSpace(memory), 64)
		if err != nil {
			return nil, err
		}

		if intMemory < yarnLimit {
			yarnLimit = intMemory * 70 / 100
			yarnLimitStr := strconv.FormatFloat(yarnLimit, 'f', -1, 64)
			return &yarnLimitStr, nil

		} else {
			yarnLimitStr := strconv.FormatFloat(yarnLimit, 'f', -1, 64)
			return &yarnLimitStr, nil
		}
	}

}

func insertYarnLimitsToEnvFile(namenode models.Namenode, datanodes []models.Datanode) error {

	var stringBuffer strings.Builder

	file, err := os.OpenFile(".env", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("erro ao abrir o arquivo .env: %v", err)
	}
	defer file.Close()

	stringBuffer.WriteString(fmt.Sprintf("master_YARN_LIMIT=%s\n", namenode.YarnLimit))

	for i, datanode := range datanodes {
		stringBuffer.WriteString(fmt.Sprintf("datanode_%d_YARN_LIMIT=%s\n", i+1, datanode.YarnLimit))

	}

	_, err = file.WriteString(stringBuffer.String())
	if err != nil {
		return fmt.Errorf("erro ao escrever no arquivo .env: %v", err)
	}

	pathPrivateKey, err := getPathPrivateKey()
	if err != nil {
		return err
	}

	for _, datanode := range datanodes {

		if datanode.IP == "" {
			continue
		}
		destination := fmt.Sprintf(
			"%s@%s:S.H.A.N.K.S/.env",
			datanode.User,
			datanode.IP,
		)

		cmd := exec.Command(
			"scp",
			"-i", pathPrivateKey,
			"-o", "IdentitiesOnly=yes",
			"-o", "StrictHostKeyChecking=accept-new",
			".env",
			destination,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf(
				"erro ao copiar o .env para o datanode %s: %v\nSaída: %s",
				datanode.Name,
				err,
				string(output),
			)
		}
	}

	return nil
}
