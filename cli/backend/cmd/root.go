package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "shanks",
	Short: "Orquestrador de cluster para processamento distribuído",
	Long: `S.H.A.N.K.S é uma ferramenta para iniciar um cluster Hadoop em um ambiente distribuído.
	Ela lê um arquivo de configuração YAML que contém informações sobre o cluster, como os nós e suas credenciais, e 
	configura os arquivos /etc/hosts de cada nó para que eles possam se comunicar entre si.
	Exemplo de uso:
  	./start --config /caminho/para/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
