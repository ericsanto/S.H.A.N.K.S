/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/usecase"
	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var cmdStart = &cobra.Command{
	Use:   "start",
	Short: "Inicia o cluster",
	Long: `S.H.A.N.K.S é uma ferramenta para iniciar um cluster Hadoop em um ambiente distribuído.
	Ela lê um arquivo de configuração YAML que contém informações sobre o cluster, como os nós e suas credenciais, e 
	configura os arquivos /etc/hosts de cada nó para que eles possam se comunicar entre si.
	Exemplo de uso:
  	./start --config /caminho/para/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfigs, err := usecase.ReadYaml(cfgFile)
		if err != nil {
			fmt.Println("❌ Não foi possível fazer a leitura do arquivo:", err)
			return
		}

		if err := usecase.CreateFileScrapingPrometheus(clusterConfigs.Cluster); err != nil {
			fmt.Println("❌ Erro ao criar arquivo de scraping do Prometheus:", err)
			return
		}

		if err := usecase.CreateDockerfile(clusterConfigs.Cluster); err != nil {
			fmt.Println("❌ Erro ao criar arquivo do Docker Compose:", err)
			return
		}

		if err := usecase.ConfigHosts(*clusterConfigs); err != nil {
			fmt.Println("❌ Erro ao configurar as máquinas via SSH:", err)
			return
		}

		if err := usecase.ConfigureYarnLimits(*clusterConfigs); err != nil {
			fmt.Println("❌ Erro ao configurar os limites do YARN:", err)
			return
		}

		if err := usecase.StartCluster(*clusterConfigs); err != nil {
			fmt.Println("❌ Erro ao iniciar os containers:", err)
			return
		}
	},
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	cmdStart.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "caminho para o arquivo yaml")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.AddCommand(cmdStart)
}
