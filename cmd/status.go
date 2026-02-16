package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/status"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show live system status snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}

		svc := status.NewService()
		result, err := svc.Run(cmd.Context(), app)
		if err != nil {
			return err
		}
		return printResult(result)
	},
}
