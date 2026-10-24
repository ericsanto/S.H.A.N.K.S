package cmd

import (
	"fmt"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/usecase"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop", // Como o usuário vai chamar no terminal
	Short: "Para a execução do cluster",
	Long:  `Este comando encerra os containers e desfaz as configurações de rede do cluster S.H.A.N.K.S.`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("🛑 Iniciando o processo de parada do cluster...")

		configCluster, err := usecase.ReadYaml(cfgFile)

		if err != nil {
			fmt.Println("❌ Erro ao ler a configuração:", err)
			return
		}

		if err := usecase.StopCluster(*configCluster); err != nil {
			return
		}
	},
}

func init() {
	// Mantém as flags que você já configurou
	stopCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "caminho para o arquivo yaml")

	// 2. Adicione o novo comando ao rootCmd
	rootCmd.AddCommand(stopCmd)
}
