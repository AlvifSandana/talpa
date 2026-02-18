package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/optimize"
)

var optimizeApply bool

var optimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Run safe optimization workflow",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		svc := optimize.NewService()
		result, err := svc.Run(cmd.Context(), app, optimize.Options{Apply: optimizeApply})
		if err != nil {
			return err
		}
		return printResult(result)
	},
}

func init() {
	optimizeCmd.Flags().BoolVar(&optimizeApply, "apply", false, "Execute optimization actions (requires --yes --confirm HIGH-RISK or --dry-run)")
}
