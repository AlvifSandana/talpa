package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/app/status"
)

var statusWatch bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show live system status snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}
		if app.Options.StatusInterval < 1 {
			return fmt.Errorf("--interval must be >= 1")
		}
		if app.Options.StatusTop < 1 {
			return fmt.Errorf("--top must be >= 1")
		}

		runs := 1
		if statusWatch {
			runs = 0
		}

		svc := status.NewService()
		for i := 0; runs == 0 || i < runs; i++ {
			result, err := svc.Run(cmd.Context(), app)
			if err != nil {
				return err
			}
			if err := printResult(result); err != nil {
				return err
			}
			if runs == 0 || i+1 < runs {
				time.Sleep(time.Duration(app.Options.StatusInterval) * time.Second)
			}
		}
		return nil
	},
}

func init() {
	statusCmd.Flags().IntVar(&opts.StatusTop, "top", 5, "Number of top processes by memory")
	statusCmd.Flags().IntVar(&opts.StatusInterval, "interval", 1, "Refresh interval in seconds")
	statusCmd.Flags().BoolVar(&statusWatch, "watch", false, "Continuously refresh status output")
}
