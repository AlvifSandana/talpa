package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/analyze"
	"talpa/internal/app/common"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze disk usage under a path",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}

		root := ""
		if len(args) == 1 {
			root = args[0]
		}

		svc := analyze.NewService()
		result, err := svc.Run(cmd.Context(), app, root)
		if err != nil {
			return err
		}
		return printResult(result)
	},
}
