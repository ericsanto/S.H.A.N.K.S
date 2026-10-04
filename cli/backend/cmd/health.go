package cmd

import (
	"fmt"
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/usecase"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health", // Como o usuário vai chamar no terminal
	Short: "Verifica a saúde do cluster",
	Long:  `Este comando verifica a saúde dos nós do cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		configCluster, err := usecase.ReadYaml(cfgFile)

		if err != nil {
			fmt.Println(err)
			return
		}

		tuiModel, err := usecase.Health(*configCluster)

		if err != nil {
			fmt.Println("Erro ao verificar saúde do cluster:", err)
			return
		}

		if _, err := tea.NewProgram(*tuiModel).Run(); err != nil {
			log.Fatal(err)
		}

		// fmt.Println("Processo de verificação de saúde concluído com sucesso.")
		// fmt.Println(health)

	},
}

func init() {
	// Mantém as flags que você já configurou
	healthCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "caminho para o arquivo yaml")

	// 2. Adicione o novo comando ao rootCmd
	rootCmd.AddCommand(healthCmd)
}
