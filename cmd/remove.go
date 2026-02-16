package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/remove"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Plan self-remove operation",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		svc := remove.NewService()
		result, err := svc.Run(cmd.Context(), app)
		if err != nil {
			return err
		}
		return printResult(result)
	},
}
