package usecase

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func ConfigureYarnLimits(configCluster models.Config) error {

	var wg sync.WaitGroup
	var mu sync.Mutex

	yarnMaster, err := configCluster.Cluster.Namenode.ConvertToYarnLimit()
	yarnMasterCPU, err := configCluster.Cluster.Namenode.ConvertCPULimit()

	if err != nil {
		return fmt.Errorf("erro ao converter o limite de memória do Yarn do Namenode: %v", err)
	}

	if err := configCluster.Cluster.Namenode.VerifiyValueMinYarnRequirements(); err != nil {
		return err
	}
	if err := validateYarnMasterGPU(&configCluster.Cluster.Namenode); err != nil {
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

	configCluster.Cluster.Namenode.YarnLimit = *validatedYarnMaster
	configCluster.Cluster.Namenode.YarnLimitCPUVcores = *validatedYarnCPU

	var yarnWorkers []*models.Datanode

	chanErrors := make(chan error, len(configCluster.Cluster.Datanodes))

	if len(configCluster.Cluster.Datanodes) > 0 {

		for _, datanode := range configCluster.Cluster.Datanodes {

			logInfo("Validando limites do Yarn para o Datanode: %s", datanode.Name)

			wg.Add(1)
			go func(datanode *models.Datanode) {

				defer wg.Done()
				if err := validateYarnGPU(datanode, pathPrivateKey); err != nil {
					chanErrors <- err
					return
				}

				yarnLimit, err := datanode.ConvertToYarnLimit()
				if err != nil {
					chanErrors <- fmt.Errorf("erro ao converter o limite de memória do Yarn do Datanode %s: %v", datanode.Name, err)
					return
				}

				yarnCPU, err := datanode.ConvertCPULimit()

				if err != nil {
					chanErrors <- err
					return
				}

				if err := datanode.VerifiyValueMinYarnRequirements(); err != nil {
					chanErrors <- err
					return
				}

				validatedYarnLimit, err := validateYarnLimits(datanode.IP, datanode.User, pathPrivateKey, yarnLimit, "datanode")
				if err != nil {
					chanErrors <- err
					return
				}

				validatedYarnCPU, err := validateYarnLimitCPUVcores(datanode.IP, datanode.User, pathPrivateKey, *yarnCPU, "datanode")

				if err != nil {
					chanErrors <- err
					return
				}

				datanode.YarnLimit = *validatedYarnLimit
				datanode.YarnLimitCPUVcores = *validatedYarnCPU

				addWorker(&mu, &yarnWorkers, datanode)

			}(datanode)

		}

	}

	wg.Wait()
	close(chanErrors)

	if err := insertYarnLimitsToEnvFile(configCluster.Cluster.Namenode, yarnWorkers); err != nil {
		return err
	}

	return nil

}

func addWorker(mu *sync.Mutex, yarnWorkers *[]*models.Datanode, datanode *models.Datanode) {

	mu.Lock()
	defer mu.Unlock()
	*yarnWorkers = append(*yarnWorkers, datanode)
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

func insertYarnLimitsToEnvFile(namenode models.Namenode, datanodes []*models.Datanode) error {
	file, err := os.OpenFile(".env", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("erro ao abrir o arquivo .env: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(buildYarnEnv(namenode, datanodes))
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

func buildYarnEnv(namenode models.Namenode, datanodes []*models.Datanode) string {
	var stringBuffer strings.Builder

	stringBuffer.WriteString(fmt.Sprintf("master_YARN_LIMIT=%s\n", namenode.YarnLimit))
	stringBuffer.WriteString(fmt.Sprintf("master_YARN_LIMIT_CPU=%s\n", namenode.YarnLimitCPUVcores))
	if namenode.GPU {
		stringBuffer.WriteString("master_GPU=true\n")
	}

	for _, datanode := range datanodes {
		stringBuffer.WriteString(fmt.Sprintf("%s_YARN_LIMIT=%s\n", datanode.Name, datanode.YarnLimit))
		stringBuffer.WriteString(fmt.Sprintf("%s_YARN_LIMIT_CPU=%s\n", datanode.Name, datanode.YarnLimitCPUVcores))

		if datanode.GPU {
			stringBuffer.WriteString(fmt.Sprintf("%s_GPU=true\n", datanode.Name))
		}
	}

	return stringBuffer.String()
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

}

func validateYarnGPU(datanode *models.Datanode, pathPrivateKey string) error {

	if datanode.GPU {
		// Além de localizar o binário, confirma que o driver consegue enumerar a
		// GPU. O código anterior comparava a saída (que termina em \n) com um
		// caminho fixo e rejeitava máquinas válidas.
		_, err := runSSHCommand(datanode.IP, "22", datanode.User, pathPrivateKey, "nvidia-smi -L")
		if err != nil {
			return fmt.Errorf("não foi possível validar a GPU NVIDIA no datanode %s: %w", datanode.Name, err)
		}

		return nil
	}

	return nil

}

func validateYarnMasterGPU(namenode *models.Namenode) error {
	if !namenode.GPU {
		return nil
	}

	output, err := exec.Command("nvidia-smi", "-L").CombinedOutput()
	if err != nil {
		return fmt.Errorf("não foi possível validar a GPU NVIDIA na master %s: %w\nSaída: %s", namenode.Name, err, strings.TrimSpace(string(output)))
	}

	return nil
}
