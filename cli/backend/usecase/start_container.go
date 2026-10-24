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

type imageVerificationResult struct {
	output string
	err    error
}

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
	// Cada fluxo (master ou worker) produz no máximo um erro. O buffer evita
	// que uma etapa bloqueie antes de conseguir sinalizar a próxima.
	errors := make(chan error, len(configCluster.Cluster.Datanodes)+1)

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
	// O cache é essencial aqui: a imagem base baixa e expande vários gigabytes.
	// Forçar --no-cache fazia toda tentativa reiniciar do zero após uma queda de
	// rede e acumulava dezenas de gigabytes de camadas sem tag nos workers.
	masterBaseImage, masterTorchFlavor := mlImageConfig(configCluster.Cluster.Namenode.GPU)
	commandDockerBuildImageMaster := fmt.Sprintf(
		"cd %s/S.H.A.N.K.S/new_cluster && docker build --build-arg TORCH_FLAVOR=%s -t %s -f Dockerfile.base .",
		home,
		masterTorchFlavor,
		masterBaseImage,
	)

	privatePathSSHKey, err := getPathPrivateKey()
	if err != nil {
		return err
	}

	logInfo("Verificando a imagem hadoop-base na master...")
	isImageBuilded := fmt.Sprintf("docker images -q %s", masterBaseImage)
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
		workerBaseImage, workerTorchFlavor := mlImageConfig(datanode.GPU)
		workerImageBuildCommand := fmt.Sprintf(
			"cd $HOME/S.H.A.N.K.S/new_cluster && docker build --build-arg TORCH_FLAVOR=%s -t %s -f Dockerfile.base .",
			workerTorchFlavor,
			workerBaseImage,
		)
		workerImageVerificationCommand := fmt.Sprintf("docker images -q %s", workerBaseImage)

		imageVerified := make(chan imageVerificationResult, 1)
		imageReady := make(chan bool, 1)

		wg.Add(1)
		go verifyImageBuildInDatanode(
			*datanode,
			errors,
			imageVerified,
			privatePathSSHKey,
			workerImageVerificationCommand,
			&wg,
		)

		wg.Add(1)
		go imageBuildDatanode(
			*datanode,
			imageVerified,
			privatePathSSHKey,
			workerImageBuildCommand,
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

	logInfo("🚧 Construindo a imagem hadoop-base na master; isso pode demorar...")

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

		logSuccess("✅ Container da master iniciado.")
		return
	} else {
		logInfo("Container da master não será iniciado porque o build falhou.")
	}

}

func verifyImageBuildInDatanode(datanode models.Datanode, errors chan<- error, resultImageBuildVerifiedDatanode chan<- imageVerificationResult, privatePathSSHKey, isImageBuilded string,
	wg *sync.WaitGroup) {

	defer wg.Done()

	logInfo("[%s] Verificando a imagem hadoop-base em %s...", datanode.Name, datanode.IP)
	outSsh, err := runSSHCommand(datanode.IP, "22", datanode.User, privatePathSSHKey, isImageBuilded)
	if err != nil {
		logFailure("[%s] Não foi possível verificar a imagem.", datanode.Name)
		errors <- fmt.Errorf("erro na verificação do build no datanode %s: %v", datanode.Name, err)
		resultImageBuildVerifiedDatanode <- imageVerificationResult{err: err}
		return
	} else if strings.TrimSpace(outSsh) == "" {
		logInfo("[%s] Imagem não encontrada; iniciando o build.", datanode.Name)
	} else {
		logSuccess("[%s] Imagem hadoop-base encontrada.", datanode.Name)
	}

	resultImageBuildVerifiedDatanode <- imageVerificationResult{output: strings.TrimSpace(outSsh)}

}

func imageBuildDatanode(datanode models.Datanode, resultImageBuildVerifiedDatanode <-chan imageVerificationResult, privatePathSSHKey, commandDockerBuildImage string, errors chan<- error,
	wg *sync.WaitGroup, resultBuidImageDatanode chan<- bool) {

	defer wg.Done()
	imageReady := false
	defer func() {
		// Toda saída precisa liberar okBuildImage; caso contrário, o comando
		// start fica bloqueado para sempre neste worker.
		resultBuidImageDatanode <- imageReady
	}()

	verification := <-resultImageBuildVerifiedDatanode

	if verification.err != nil {
		return
	}

	if verification.output == "" {
		logInfo("[%s] Construindo a imagem hadoop-base; isso pode demorar...", datanode.Name)

		outBuild, err := runSSHCommand(datanode.IP, "22", datanode.User, privatePathSSHKey, commandDockerBuildImage)

		if err != nil {
			logFailure("[%s] Não foi possível construir a imagem.", datanode.Name)
			errors <- fmt.Errorf("erro ao fazer o build da imagem no datanode %s: %v\nLogs: %s", datanode.Name, err, outBuild)
			return
		}

		imageReady = true
		logSuccess("[%s] Build da imagem concluído.", datanode.Name)
	} else {
		imageReady = true
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
