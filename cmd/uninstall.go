package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/uninstall"
)

var uninstallApply bool
var uninstallTargets []string

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall app and Talpa-related leftovers safely",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		svc := uninstall.NewService()
		result, err := svc.Run(cmd.Context(), app, uninstall.Options{Apply: uninstallApply, Targets: uninstallTargets})
		if err != nil {
			return err
		}
		return printResult(result)
	},
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallApply, "apply", false, "Execute uninstall actions (requires --yes --confirm HIGH-RISK or --dry-run)")
	uninstallCmd.Flags().StringSliceVar(&uninstallTargets, "target", nil, "Explicit uninstall target in backend:name format (backend: apt|dnf|pacman|zypper|snap|flatpak)")
}
