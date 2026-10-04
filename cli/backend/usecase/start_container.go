package usecase

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

var logMu sync.Mutex

func logInfo(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Printf("\033[1;34m[INFO]\033[0m "+format+"\n", args...)
}

func logSuccess(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Printf("\033[1;32m[OK]\033[0m "+format+"\n", args...)
}

func logFailure(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Printf("\033[1;31m[FAIL]\033[0m "+format+"\n", args...)
}

func okBuildImage(resultBuidImageDatanode <-chan bool, errors chan<- error, wg *sync.WaitGroup, datanode models.Datanode, pathPrivateSSHKey, commandWorkers string) {

	defer wg.Done()

	if <-resultBuidImageDatanode {
		wg.Add(1)
		go startContainerDatanode(datanode, pathPrivateSSHKey, commandWorkers, errors, wg)
		return
	} else {
		logInfo("[%s] Container não será iniciado porque o build falhou.", datanode.Name)
		return
	}
}

func startSpinner(message string, done <-chan bool, fail chan bool) {

	frames := []string{"|", "/", "-", "\\"}
	i := 0

	for {
		select {
		case <-done:

			fmt.Printf("\r%s [ \033[32mOK\033[0m ]      \n", message)
			return

		case <-fail:
			fmt.Printf("\r%s [ \033[31mFAIL\033[0m ]          \n", message)

		default:

			fmt.Printf("\r%s %s", message, frames[i])

			i = (i + 1) % len(frames)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func StartCluster(configCluster models.Config) error {
	fmt.Println()
	fmt.Println()

	fmt.Println("\033[1;31m  ____   _   _      _      _   _   _  __  ____  \033[1;33m         ____      \033[0m")
	fmt.Println("\033[1;31m / ___| | | | |    / \\    | \\ | | | |/ / / ___| \033[1;33m       /      \\    \033[0m")
	fmt.Println("\033[1;31m \\___ \\ | |_| |   / _ \\   |  \\| | | ' /  \\___ \\ \033[1;31m      |========|   \033[0m")
	fmt.Println("\033[1;31m  ___) ||  _  |_ / ___ \\ _| |\\  |_| . \\ _ ___) |\033[1;33m    _/__|____|__\\_ \033[0m")
	fmt.Println("\033[1;31m |____(_)_| |_(_)_/   \\_(_)_| \\_(_)_|\\_(_)____/ \033[1;33m   (______________)\033[0m")

	fmt.Println()
	fmt.Println()

	done := make(chan bool)
	fail := make(chan bool)
	isImageBuildOK := make(chan bool, 1)
	errors := make(chan error)

	// resultImageBuildVerifiedMaster := make(chan bool, 1)

	var wg sync.WaitGroup

	go startSpinner("Iniciando cluster...", done, fail)
	time.Sleep(time.Second * 3)
	done <- true

	home, err := getHomeDir()

	if err != nil {
		return err
	}

	commandMaster := fmt.Sprintf("cd %s/S.H.A.N.K.S/new_cluster/master && docker compose -f 'docker-compose.master.yml' up -d --build", home)
	commandWorkers := "cd $HOME/S.H.A.N.K.S/new_cluster/worker && docker compose -f 'docker-compose.worker.yml' up -d --build"
	commandDockerBuildImageMaster := fmt.Sprintf("cd %s/S.H.A.N.K.S/new_cluster && docker build --no-cache -t hadoop-base -f Dockerfile.base .", home)
	commandDockerBuildWorkers := "cd $HOME/S.H.A.N.K.S/new_cluster && docker build --no-cache -t hadoop-base -f Dockerfile.base ."

	privatePathSSHKey, err := getPathPrivateKey()
	if err != nil {
		return err
	}

	logInfo("Verificando a imagem hadoop-base na master...")
	isImageBuilded := "docker images -q hadoop-base"
	out, err := exec.Command("bash", "-c", isImageBuilded).Output()

	if err != nil {

		return fmt.Errorf("erro ao verificar se a imagem docker existe no host: %v", err)
	}

	resultFormated := strings.TrimSpace(string(out))

	if resultFormated == "" {
		logInfo("Imagem hadoop-base não encontrada na master; iniciando o build.")
		//FAZ O BUILD DA IMAGEM CASO NAO TENHA
		wg.Add(1)
		go buildImage(commandDockerBuildImageMaster, errors, &wg, isImageBuildOK)

	} else {
		logSuccess("Imagem hadoop-base encontrada na master.")
		isImageBuildOK <- true
	}

	//ESPERA O SINAL DO BUILD DA IMAGEM
	wg.Add(1)
	go startContainer(commandMaster, isImageBuildOK, errors, &wg)

	for _, datanode := range configCluster.Cluster.Datanodes {

		imageVerified := make(chan string, 1)
		imageReady := make(chan bool, 1)

		wg.Add(1)
		go verifyImageBuildInDatanode(
			*datanode,
			errors,
			imageVerified,
			privatePathSSHKey,
			isImageBuilded,
			&wg,
		)

		wg.Add(1)
		go imageBuildDatanode(
			*datanode,
			imageVerified,
			privatePathSSHKey,
			commandDockerBuildWorkers,
			errors,
			&wg,
			imageReady,
		)

		wg.Add(1)
		go okBuildImage(
			imageReady,
			errors,
			&wg,
			*datanode,
			privatePathSSHKey,
			commandWorkers,
		)

	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	var firstError error
	for err := range errors {
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	if firstError != nil {
		return firstError
	}

	fmt.Println("\n ☠️🗡️ Cluster Iniciado com Sucesso!")
	return nil
}

func buildImage(commandImageBuilder string, errors chan<- error, wg *sync.WaitGroup, isImageBuildOK chan<- bool) {

	logInfo("Construindo a imagem hadoop-base na master; isso pode demorar...")

	defer wg.Done()

	outBuild, errBuild := exec.Command("bash", "-c", commandImageBuilder).CombinedOutput()

	if errBuild != nil {
		logFailure("Não foi possível construir a imagem na master.")
		isImageBuildOK <- false
		errors <- fmt.Errorf("erro ao fazer o build da imagem na master: %v\nLogs: %s", errBuild, string(outBuild))
		return
	}

	logSuccess("Build da imagem na master concluído.")
	isImageBuildOK <- true

}

func startContainer(commandMaster string, isBuildImageOK <-chan bool, errors chan<- error, wg *sync.WaitGroup) {

	defer wg.Done()

	if <-isBuildImageOK {
		logInfo("Iniciando o container da master...")

		output, err := exec.Command("bash", "-c", commandMaster).CombinedOutput()
		if err != nil {
			logFailure("Não foi possível iniciar o container da master.")
			errors <- fmt.Errorf("erro ao iniciar o container da master: %v\nLogs: %s", err, string(output))
			return
		}

		logSuccess("Container da master iniciado.")
		return
	} else {
		logInfo("Container da master não será iniciado porque o build falhou.")
	}

}

func verifyImageBuildInDatanode(datanode models.Datanode, errors chan<- error, resultImageBuildVerifiedDatanode chan<- string, privatePathSSHKey, isImageBuilded string,
	wg *sync.WaitGroup) {

	defer wg.Done()

	logInfo("[%s] Verificando a imagem hadoop-base em %s...", datanode.Name, datanode.IP)
	outSsh, err := runSSHCommand(datanode.IP, "22", datanode.User, privatePathSSHKey, isImageBuilded)
	if err != nil {
		logFailure("[%s] Não foi possível verificar a imagem.", datanode.Name)
		errors <- fmt.Errorf("erro na verificação do build no datanode %s: %v", datanode.Name, err)
	} else if strings.TrimSpace(outSsh) == "" {
		logInfo("[%s] Imagem não encontrada; iniciando o build.", datanode.Name)
	} else {
		logSuccess("[%s] Imagem hadoop-base encontrada.", datanode.Name)
	}

	resultImageBuildVerifiedDatanode <- strings.TrimSpace(outSsh)

}

func imageBuildDatanode(datanode models.Datanode, resultImageBuildVerifiedDatanode <-chan string, privatePathSSHKey, commandDockerBuildImage string, errors chan<- error,
	wg *sync.WaitGroup, resultBuidImageDatanode chan<- bool) {

	defer wg.Done()
	output := <-resultImageBuildVerifiedDatanode

	if output == "" {
		logInfo("[%s] Construindo a imagem hadoop-base; isso pode demorar...", datanode.Name)

		outBuild, err := runSSHCommand(datanode.IP, "22", datanode.User, privatePathSSHKey, commandDockerBuildImage)

		if err != nil {
			logFailure("[%s] Não foi possível construir a imagem.", datanode.Name)
			errors <- fmt.Errorf("erro ao fazer o build da imagem no datanode %s: %v\nLogs: %s", datanode.Name, err, outBuild)
			return
		}

		resultBuidImageDatanode <- true

		logSuccess("[%s] Build da imagem concluído.", datanode.Name)
	} else {
		resultBuidImageDatanode <- true
	}
}

func startContainerDatanode(datanode models.Datanode, privatePathSSHKey, commandWorkers string, errors chan<- error, wg *sync.WaitGroup) {

	defer wg.Done()

	logInfo("[%s] Iniciando o container em %s...", datanode.Name, datanode.IP)
	_, err := runSSHCommand(datanode.IP, "22", datanode.User, privatePathSSHKey, commandWorkers)
	if err != nil {
		logFailure("[%s] Não foi possível iniciar o container.", datanode.Name)
		errors <- fmt.Errorf("erro ao subir container no datanode %s: %v", datanode.Name, err)
		return
	}

	logSuccess("[%s] Container iniciado.", datanode.Name)
}
