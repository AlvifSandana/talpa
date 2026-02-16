package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/installer"
)

var installerApply bool

var installerCmd = &cobra.Command{
	Use:   "installer",
	Short: "Scaffold installer artifact cleanup",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		svc := installer.NewService()
		result, err := svc.Run(cmd.Context(), app, installer.Options{Apply: installerApply})
		if err != nil {
			return err
		}
		return printResult(result)
	},
}

func init() {
	installerCmd.Flags().BoolVar(&installerApply, "apply", false, "Execute installer cleanup actions (requires --yes or --dry-run)")
}
