package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"talpa/internal/app/analyze"
	"talpa/internal/app/common"
)

var analyzeDepth int
var analyzeLimit int
var analyzeSort string
var analyzeMinSize int64
var analyzeQuery string
var analyzeOnlyCandidates bool
var analyzeAction string

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze disk usage under a path",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		if analyzeDepth < 1 {
			return fmt.Errorf("--depth must be >= 1")
		}
		if analyzeLimit < 0 {
			return fmt.Errorf("--limit must be >= 0")
		}
		if analyzeMinSize < 0 {
			return fmt.Errorf("--min-size must be >= 0")
		}
		switch analyzeSort {
		case "size", "path", "mtime":
		default:
			return fmt.Errorf("--sort must be one of: size, path, mtime")
		}
		switch analyzeAction {
		case "inspect", "trash", "delete":
		default:
			return fmt.Errorf("--action must be one of: inspect, trash, delete")
		}

		root := ""
		if len(args) == 1 {
			root = args[0]
		}

		svc := analyze.NewService()
		result, err := svc.Run(cmd.Context(), app, root, analyze.Options{
			Depth:          analyzeDepth,
			Limit:          analyzeLimit,
			SortBy:         analyzeSort,
			MinSizeBytes:   analyzeMinSize,
			Query:          analyzeQuery,
			OnlyCandidates: analyzeOnlyCandidates,
			Action:         analyzeAction,
		})
		if err != nil {
			return err
		}
		return printResult(result)
	},
}

func init() {
	analyzeCmd.Flags().IntVar(&analyzeDepth, "depth", 4, "Maximum scan depth")
	analyzeCmd.Flags().IntVar(&analyzeLimit, "limit", 50, "Maximum number of nodes (0 = unlimited)")
	analyzeCmd.Flags().StringVar(&analyzeSort, "sort", "size", "Sort by: size, path, mtime")
	analyzeCmd.Flags().Int64Var(&analyzeMinSize, "min-size", 0, "Minimum item size in bytes")
	analyzeCmd.Flags().StringVar(&analyzeQuery, "query", "", "Filter item paths by substring")
	analyzeCmd.Flags().BoolVar(&analyzeOnlyCandidates, "only-candidates", false, "Show only cleanup candidates")
	analyzeCmd.Flags().StringVar(&analyzeAction, "action", "inspect", "Action mode: inspect, trash, delete")
}
