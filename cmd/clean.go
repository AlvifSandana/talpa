package cmd

import (
	"github.com/spf13/cobra"

	"talpa/internal/app/clean"
	"talpa/internal/app/common"
)

var cleanSystem bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean safe cache and temp files",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := common.FromCommand(cmd)
		if err != nil {
			return err
		}

		svc := clean.NewService()
		result, err := svc.Run(cmd.Context(), app, clean.Options{System: cleanSystem})
		if err != nil {
			return err
		}
		return printResult(result)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanSystem, "system", false, "Include opt-in system-level cleanup candidates")
}
