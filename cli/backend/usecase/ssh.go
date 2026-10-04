package usecase

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func runSSHCommand(ip string, port string, username string, privateKeyPath string, command string) (string, error) {

	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("não foi possível ler a chave privada: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("formato de chave privada inválido: %v", err)
	}

	// A autenticação é feita utilizando a chave privada do usuário em que a cli está sendo executada
	// Para a autenticação funcionar, é necessário que a chave pública correspondente à chave privada esteja presente no arquivo ~/.ssh/authorized_keys do usuário remoto.
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second, // Desiste se o Worker estiver desligado
	}

	address := fmt.Sprintf("%s:%s", ip, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return "", fmt.Errorf("falha ao conectar no IP %s: %v", ip, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("falha ao abrir sessão ssh: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("erro ao rodar comando '%s': %v\nSaída: %s", command, err, string(output))
	}

	return string(output), nil
}

func runSSHCommandBuffer(ip string, port string, username string, privateKeyPath string, command string, stdin string) (string, error) {

	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("não foi possível ler a chave privada: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("formato de chave privada inválido: %v", err)
	}

	// A autenticação é feita utilizando a chave privada do usuário em que a cli está sendo executada
	// Para a autenticação funcionar, é necessário que a chave pública correspondente à chave privada esteja presente no arquivo ~/.ssh/authorized_keys do usuário remoto.
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second, // Desiste se o Worker estiver desligado
	}

	address := fmt.Sprintf("%s:%s", ip, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return "", fmt.Errorf("falha ao conectar no IP %s: %v", ip, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("falha ao abrir sessão ssh: %v", err)
	}
	defer session.Close()

	session.Stdin = strings.NewReader(stdin)

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("erro ao rodar comando '%s': %v\nSaída: %s", command, err, string(output))
	}

	return string(output), nil
}

// PQ EU PRECISO PEGAR O CAMINHO DA CHAVE PRIVADA DO USUÁRIO ORIGINAL DO SUDO?
// R: Porque quando o usuário executa o comando com sudo, o diretório HOME muda para /root
//
//	e a chave privada do usuário original não está lá.
//
// Então, precisamos pegar o caminho da chave privada do usuário que executou o sudo, não do root.
func getPathPrivateKey() (string, error) {

	home, err := getHomeDir()

	if err != nil {
		return "", err
	}

	caminhoCompleto := filepath.Join(home, ".ssh", "id_rsa")

	return caminhoCompleto, nil
}

func getHomeDir() (string, error) {
	var home string

	sudoUser := os.Getenv("SUDO_USER")

	if sudoUser != "" {

		// O comando foi executado com sudo, então precisamos pegar o diretório HOME do usuário original
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("erro ao buscar o usuário original do sudo: %v", err)
		}
		home = u.HomeDir
	} else {

		h, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("erro ao localizar o diretório HOME: %v", err)
		}
		home = h
	}

	return home, nil
}
