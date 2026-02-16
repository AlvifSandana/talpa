package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/update"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Plan self-update operation",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		svc := update.NewService()
		result, err := svc.Run(cmd.Context(), app)
		if err != nil {
			return err
		}
		return printResult(result)
	},
}
