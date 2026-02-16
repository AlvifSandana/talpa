package uninstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/domain/safety"
)

type Service struct{}

type Options struct {
	Apply bool
}

var (
	osUserHomeDir      = os.UserHomeDir
	osExecutable       = os.Executable
	osStat             = os.Stat
	safeDelete         = safety.SafeDelete
	pathValidateSystem = common.ValidateSystemScopePath
)

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, opts Options) (model.CommandResult, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return model.CommandResult{}, err
	}
	exe, exeErr := osExecutable()
	binaryTargets := uninstallBinaryTargets(home, exe, exeErr)

	items := make([]model.CandidateItem, 0, len(binaryTargets)+2)
	for i, target := range binaryTargets {
		id := fmt.Sprintf("uninstall-%d", i+1)
		ruleID := "uninstall.binary"
		if i > 0 {
			ruleID = "uninstall.binary.system"
		}
		items = append(items, newPlanItem(id, ruleID, target, "app_binary", model.RiskHigh))
	}
	items = append(items,
		newPlanItem("uninstall-3", "uninstall.config", filepath.Join(home, ".config", "talpa"), "config", model.RiskMedium),
		newPlanItem("uninstall-4", "uninstall.cache", filepath.Join(home, ".cache", "talpa"), "cache", model.RiskLow),
	)

	errCount := 0
	if opts.Apply {
		if err := common.RequireConfirmationOrDryRun(app.Options, "uninstall"); err != nil {
			return model.CommandResult{}, err
		}
		if !app.Options.DryRun {
			for i := range items {
				entry := model.OperationLogEntry{
					Timestamp: time.Now().UTC(),
					PlanID:    "plan-uninstall",
					Command:   "uninstall",
					Action:    "delete",
					Path:      items[i].Path,
					RuleID:    items[i].RuleID,
					Category:  items[i].Category,
					Risk:      string(items[i].Risk),
					DryRun:    false,
				}
				if err := pathValidateSystem(items[i].Path, app.Whitelist); err != nil {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}
				if _, err := osStat(items[i].Path); errors.Is(err, os.ErrNotExist) {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}
				if err := safeDelete(items[i].Path, nil, app.Whitelist, false); err != nil {
					items[i].Result = "error"
					errCount++
				} else {
					items[i].Result = "deleted"
				}
				entry.Result = items[i].Result
				if err := app.Logger.Log(ctx, entry); err != nil {
					errCount++
				}
			}
		}
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "uninstall",
		Timestamp:     time.Now().UTC(),
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:    len(items),
			ItemsSelected: len(items),
			Errors:        errCount,
		},
		Items: items,
	}, nil
}

func uninstallBinaryTargets(home, executable string, executableErr error) []string {
	targets := []string{
		filepath.Join(home, ".local", "bin", "talpa"),
		"/usr/local/bin/talpa",
	}

	if executableErr == nil {
		trusted, normalized := trustedExecutablePath(executable, targets)
		if trusted {
			targets = append(targets, normalized)
		}
	}

	seen := make(map[string]struct{}, len(targets))
	uniq := make([]string, 0, len(targets))
	for _, p := range targets {
		n := filepath.Clean(p)
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		uniq = append(uniq, n)
	}

	return uniq
}

func trustedExecutablePath(executable string, canonical []string) (bool, string) {
	normalized := filepath.Clean(executable)
	if normalized == "" {
		return false, ""
	}
	if strings.HasPrefix(normalized, os.TempDir()+string(filepath.Separator)) {
		return false, ""
	}
	for _, p := range canonical {
		if normalized == filepath.Clean(p) {
			return true, normalized
		}
	}
	return false, ""
}

func newPlanItem(id, ruleID, path, category string, risk model.RiskLevel) model.CandidateItem {
	return model.CandidateItem{
		ID:           id,
		RuleID:       ruleID,
		Path:         path,
		Category:     category,
		Risk:         risk,
		Selected:     true,
		RequiresRoot: false,
		LastModified: time.Now().UTC(),
		Result:       "planned",
	}
}
