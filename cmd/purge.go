package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/purge"
)

var purgePaths string

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge project build artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}

		var paths []string
		if strings.TrimSpace(purgePaths) != "" {
			for _, p := range strings.Split(purgePaths, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					paths = append(paths, p)
				}
			}
		}

		svc := purge.NewService()
		result, err := svc.Run(cmd.Context(), app, paths)
		if err != nil {
			return err
		}
		return printResult(result)
	},
}

func init() {
	purgeCmd.Flags().StringVar(&purgePaths, "paths", "", "Comma-separated paths to scan")
}
