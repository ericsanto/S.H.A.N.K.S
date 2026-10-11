package cmd

import (
	"fmt"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/usecase"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune", // Como o usuário vai chamar no terminal
	Short: "Remove os containers e imagens não utilizadas",
	Long:  `Este comando remove os containers e imagens não utilizadas do sistema.`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("🧹 Iniciando o processo de remoção de recursos não utilizados...")

		configCluster, err := usecase.ReadYaml(cfgFile)

		if err != nil {
			fmt.Println("❌ Erro ao ler a configuração:", err)
			return
		}

		if err := usecase.PruneCluster(configCluster); err != nil {
			fmt.Println("❌ Erro ao remover recursos não utilizados:", err)
			return
		}

		fmt.Println("✅ Processo de remoção de recursos não utilizados concluído com sucesso.")
	},
}

func init() {
	// Mantém as flags que você já configurou
	pruneCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "caminho para o arquivo yaml")

	// 2. Adicione o novo comando ao rootCmd
	rootCmd.AddCommand(pruneCmd)
}
