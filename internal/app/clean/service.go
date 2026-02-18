package clean

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/domain/rules"
	"talpa/internal/domain/safety"
)

type Service struct{}

var getEUID = os.Geteuid
var cleanReadDir = os.ReadDir
var cleanSafeDelete = safety.SafeDelete

type Options struct {
	System bool
}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, opts Options) (model.CommandResult, error) {
	start := time.Now()
	home, err := os.UserHomeDir()
	if err != nil {
		return model.CommandResult{}, err
	}

	ruleSet := rules.ExistingCleanRules(home, opts.System)

	items := make([]model.CandidateItem, 0, len(ruleSet))
	selected := 0
	var estimate int64
	errCount := 0
	requiresHighRiskConfirm := false
	isRoot := getEUID() == 0

	for i, rule := range ruleSet {
		p := rule.Pattern
		size := dirSize(p)
		allowedRoots := cleanAllowedRoots(rule, home)
		item := model.CandidateItem{
			ID:           "clean-" + strconv.Itoa(i+1),
			RuleID:       rule.ID,
			Path:         p,
			SizeBytes:    size,
			LastModified: time.Now().UTC(),
			Category:     rule.Category,
			Risk:         rule.Risk,
			Selected:     true,
			RequiresRoot: rule.RequiresRoot,
			Result:       "planned",
		}

		if rule.RequiresRoot && !isRoot {
			item.Selected = false
			item.Result = "skipped"
		} else if err := safety.ValidatePath(p, allowedRoots, cleanWhitelistForPath(app.Whitelist, p)); err != nil {
			item.Selected = false
			item.Result = "skipped"
			errCount++
		} else if rule.Risk == model.RiskHigh {
			requiresHighRiskConfirm = true
		} else {
			selected++
			estimate += size
		}

		if item.Selected && rule.Risk == model.RiskHigh {
			selected++
			estimate += size
		}

		items = append(items, item)
	}

	if !app.Options.DryRun {
		if requiresHighRiskConfirm {
			if err := common.RequireHighRiskConfirmationOrDryRun(app.Options, "clean"); err != nil {
				return model.CommandResult{}, err
			}
		} else if !app.Options.Yes {
			return model.CommandResult{}, errors.New("confirmation required for clean: use --yes or --dry-run")
		}
		for i := range items {
			if !items[i].Selected {
				continue
			}
			err := deleteCleanTarget(items[i].Path, cleanAllowedRootsByPath(items[i].Path, home), cleanWhitelistForPath(app.Whitelist, items[i].Path), false)
			if err != nil {
				items[i].Result = "error"
				errCount++
			} else {
				items[i].Result = "deleted"
			}

			if err := app.Logger.Log(ctx, model.OperationLogEntry{
				Timestamp: time.Now().UTC(),
				PlanID:    "plan-clean",
				Command:   "clean",
				Action:    "delete",
				Path:      items[i].Path,
				RuleID:    items[i].RuleID,
				Category:  items[i].Category,
				SizeBytes: items[i].SizeBytes,
				Risk:      string(items[i].Risk),
				Result:    items[i].Result,
				DryRun:    false,
			}); err != nil {
				errCount++
			}
		}
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "clean",
		Timestamp:     time.Now().UTC(),
		DurationMS:    time.Since(start).Milliseconds(),
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:          len(items),
			ItemsSelected:       selected,
			EstimatedFreedBytes: estimate,
			Errors:              errCount,
		},
		Items: items,
	}, nil
}

func cleanAllowedRoots(rule model.Rule, home string) []string {
	if strings.HasPrefix(rule.ID, "clean.system.") {
		return cleanAllowedRootsByPath(rule.Pattern, home)
	}
	return []string{home}
}

func cleanAllowedRootsByPath(path, home string) []string {
	n := filepath.Clean(path)
	if n == filepath.Clean("/tmp") || strings.HasPrefix(n, "/tmp/") {
		return []string{"/tmp"}
	}
	if n == filepath.Clean("/var/tmp") || strings.HasPrefix(n, "/var/tmp/") {
		return []string{"/var/tmp"}
	}
	if n == filepath.Clean("/var/cache") || strings.HasPrefix(n, "/var/cache/") {
		return []string{"/var/cache"}
	}
	if n == filepath.Clean("/var/log/journal") || strings.HasPrefix(n, "/var/log/journal/") {
		return []string{"/var/log/journal"}
	}
	return []string{home}
}

func cleanWhitelistForPath(base []string, path string) []string {
	out := make([]string, 0, len(base)+1)
	out = append(out, base...)
	n := filepath.Clean(path)
	if strings.HasPrefix(n, "/tmp/") || n == "/tmp" || strings.HasPrefix(n, "/var/tmp/") || n == "/var/tmp" || strings.HasPrefix(n, "/var/cache/") || n == "/var/cache" || strings.HasPrefix(n, "/var/log/journal/") || n == "/var/log/journal" {
		out = append(out, n)
	}
	return out
}

func deleteCleanTarget(path string, allowedRoots []string, whitelist []string, dryRun bool) error {
	n := filepath.Clean(path)
	if n == "/tmp" || n == "/var/tmp" || n == "/var/cache/apt" || n == "/var/cache/dnf" || n == "/var/cache/pacman" || n == "/var/cache/zypp" || n == "/var/log/journal" {
		entries, err := cleanReadDir(n)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		for _, e := range entries {
			target := filepath.Join(n, e.Name())
			if err := cleanSafeDelete(target, allowedRoots, whitelist, dryRun); err != nil {
				return err
			}
		}
		return nil
	}
	return cleanSafeDelete(path, allowedRoots, whitelist, dryRun)
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
