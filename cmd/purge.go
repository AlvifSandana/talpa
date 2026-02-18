package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/purge"
)

var purgePaths string
var purgeDepth int
var purgeRecentDays int

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge project build artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		if err := validatePurgeFlags(purgeDepth, purgeRecentDays); err != nil {
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
		result, err := svc.Run(cmd.Context(), app, paths, purge.Options{
			MaxDepth:   purgeDepth,
			RecentDays: purgeRecentDays,
		})
		if err != nil {
			return err
		}
		return printResult(result)
	},
}

func init() {
	purgeCmd.Flags().StringVar(&purgePaths, "paths", "", "Comma-separated paths to scan")
	purgeCmd.Flags().IntVar(&purgeDepth, "depth", 4, "Maximum scan depth for artifact discovery")
	purgeCmd.Flags().IntVar(&purgeRecentDays, "recent-days", 7, "Treat artifacts modified within N days as recent and skip by default")
}

func validatePurgeFlags(depth, recentDays int) error {
	if depth < 1 {
		return fmt.Errorf("--depth must be >= 1")
	}
	if recentDays < 1 {
		return fmt.Errorf("--recent-days must be >= 1")
	}
	return nil
}
