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
	yarnMasterCPU, err := configCluster.Cluster.Namenode.ConvertCPULimit()

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

	validatedYarnCPU, err := validateYarnLimitCPUVcores(configCluster.Cluster.Namenode.IP, configCluster.Cluster.Namenode.User, pathPrivateKey, *yarnMasterCPU, "namenode")
	if err != nil {
		return err
	}

	fmt.Println("ValidatedYarnCPU: ", *validatedYarnCPU)

	configCluster.Cluster.Namenode.YarnLimit = *validatedYarnMaster
	configCluster.Cluster.Namenode.YarnLimitCPUVcores = *validatedYarnCPU

	var yarnWorkers []models.Datanode

	fmt.Println(len((configCluster.Cluster.Datanodes)))

	if len(configCluster.Cluster.Datanodes) > 0 {
		yarnWorkers = make([]models.Datanode, len(configCluster.Cluster.Datanodes)-1)
		for _, datanode := range configCluster.Cluster.Datanodes {

			if err := validateYarnGPU(datanode, pathPrivateKey); err != nil {
				return err
			}

			yarnLimit, err := datanode.ConvertToYarnLimit()
			if err != nil {
				return fmt.Errorf("erro ao converter o limite de memória do Yarn do Datanode %s: %v", datanode.Name, err)
			}

			yarnCPU, err := datanode.ConvertCPULimit()

			if err != nil {
				return err
			}

			if err := datanode.VerifiyValueMinYarnRequirements(); err != nil {
				return err
			}

			validatedYarnLimit, err := validateYarnLimits(datanode.IP, datanode.User, pathPrivateKey, yarnLimit, "datanode")
			if err != nil {
				return err
			}

			validatedYarnCPU, err := validateYarnLimitCPUVcores(datanode.IP, datanode.User, pathPrivateKey, *yarnCPU, "datanode")

			if err != nil {
				return err
			}

			datanode.YarnLimit = *validatedYarnLimit
			datanode.YarnLimitCPUVcores = *validatedYarnCPU
			yarnWorkers = append(yarnWorkers, datanode)
		}

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
	stringBuffer.WriteString(fmt.Sprintf("master_YARN_LIMIT_CPU=%s\n", namenode.YarnLimitCPUVcores))

	for i, datanode := range datanodes {
		stringBuffer.WriteString(fmt.Sprintf("datanode_%d_YARN_LIMIT=%s\n", i+1, datanode.YarnLimit))
		stringBuffer.WriteString(fmt.Sprintf("datanode_%d_YARN_LIMIT_CPU=%s\n", i+1, datanode.YarnLimitCPUVcores))

		if datanode.GPU {
			stringBuffer.WriteString(fmt.Sprintf("datanode_%d_GPU=%t\n", i+1, datanode.GPU))
		}

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

func validateYarnLimitCPUVcores(ip, user, pathPrivateKey string, cpu int, config string) (*string, error) {

	if config == "namenode" {
		result, err := exec.Command("bash", "-c", "nproc").Output()

		if err != nil {
			return nil, err
		}

		cpuF := strings.Replace(string(result), "\n", "", 1)

		quantitityCPUMachine, err := strconv.Atoi(string(cpuF))

		if err != nil {
			return nil, err
		}

		if cpu > quantitityCPUMachine {
			logInfo("Quantidade de Cpu alocada é maior que a quantidade suportada pela máquina física ", config)
			cpu = quantitityCPUMachine / 2
			cpuConverted := strconv.Itoa(cpu)
			return &cpuConverted, nil
		} else {
			cpuConverted := strconv.Itoa(cpu)
			return &cpuConverted, nil
		}
	} else {
		result, err := runSSHCommand(ip, "22", user, pathPrivateKey, "nproc")
		if err != nil {
			return nil, err
		}

		cpuF := strings.Replace(string(result), "\n", "", 1)

		cpuMachine, err := strconv.Atoi(cpuF)
		if err != nil {
			return nil, err
		}

		if cpu > cpuMachine {
			logInfo("Quantidade de Cpu alocada é maior que a quantidade suportada pela máquina física ", config)
			cpu = cpuMachine / 2
			cpuConverted := strconv.Itoa(cpu)
			return &cpuConverted, nil
		} else {
			cpuConverted := strconv.Itoa(cpu)
			return &cpuConverted, nil
		}

	}

	return nil, nil

}

func validateYarnGPU(datanode models.Datanode, pathPrivateKey string) error {

	if datanode.GPU {

		command := "command -v nvidia-smi"

		result, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, command)

		if err != nil {
			return fmt.Errorf("erro ao tentar validar driver nvidia")
		}

		if result != "/usr/bin/nvidia-smi" {
			return fmt.Errorf("erro ao alocar GPU. A máquina não contém os drivers necessários")
		}

		return nil
	}

	return nil

}
