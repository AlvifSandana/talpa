package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"talpa/internal/app/common"
	"talpa/internal/infra/config"
	"talpa/internal/infra/logging"
)

var opts common.GlobalOptions

var rootCmd = &cobra.Command{
	Use:   "talpa",
	Short: "Talpa is a Linux cleanup and analysis CLI",
	Long:  "Talpa helps clean caches, analyze disk usage, purge project artifacts, and monitor system status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return cmd.Help()
		}
		if isInteractiveTerminal() {
			return runInteractiveMenu()
		}
		return cmd.Help()
	},
}

func Execute() error {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		appCtx, err := buildAppContext(ctx)
		if err != nil {
			return err
		}
		cmd.SetContext(context.WithValue(ctx, common.ContextKeyApp, appCtx))
		return nil
	}

	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&opts.DryRun, "dry-run", false, "Preview actions without modifying files")
	rootCmd.PersistentFlags().BoolVar(&opts.Debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&opts.Yes, "yes", false, "Auto-confirm actions in non-interactive mode")
	rootCmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&opts.NoOpLog, "no-oplog", false, "Disable operation log")

	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(purgeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(installerCmd)
	rootCmd.AddCommand(optimizeCmd)
}

func printResult(v any) error {
	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}

	if line, ok := v.(fmt.Stringer); ok {
		fmt.Println(line.String())
		return nil
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func buildAppContext(ctx context.Context) (*common.AppContext, error) {
	store := config.NewStore()
	whitelist, err := store.LoadWhitelist(ctx)
	if err != nil {
		return nil, err
	}

	oplogDisabled := opts.NoOpLog || os.Getenv("TALPA_NO_OPLOG") == "1"
	oplog, err := logging.NewOperationLogger(ctx, oplogDisabled)
	if err != nil {
		oplog = logging.NewNoopLogger()
	}

	return &common.AppContext{
		Options:   opts,
		Whitelist: whitelist,
		Logger:    oplog,
	}, nil
}

func isInteractiveTerminal() bool {
	in, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	out, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return shouldUseInteractive(in.Mode(), out.Mode(), os.Getenv("TERM"))
}

func shouldUseInteractive(stdinMode os.FileMode, stdoutMode os.FileMode, term string) bool {
	if !isCharDevice(stdinMode) || !isCharDevice(stdoutMode) {
		return false
	}
	return !isDumbTerm(term)
}

func isCharDevice(mode os.FileMode) bool {
	return (mode & os.ModeCharDevice) != 0
}

func isDumbTerm(term string) bool {
	normalized := strings.ToLower(strings.TrimSpace(term))
	return normalized == "" || normalized == "dumb"
}

func runSelf(args ...string) error {
	b := exec.Command(os.Args[0], args...)
	b.Stdout = os.Stdout
	b.Stderr = os.Stderr
	b.Stdin = os.Stdin
	return b.Run()
}
