package purge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/domain/rules"
	"talpa/internal/domain/safety"
)

type Service struct{}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, paths []string) (model.CommandResult, error) {
	start := time.Now()
	if len(paths) == 0 {
		paths = defaultPaths()
	}

	ruleSet := rules.PurgeArtifactRules()
	ruleByName := make(map[string]model.Rule, len(ruleSet))
	for _, r := range ruleSet {
		ruleByName[r.Pattern] = r
	}

	home, _ := os.UserHomeDir()
	items := make([]model.CandidateItem, 0, 64)
	selected := 0
	var estimate int64
	errCount := 0

	for _, root := range paths {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || !d.IsDir() {
				return nil
			}

			name := d.Name()
			rule, ok := ruleByName[name]
			if !ok {
				return nil
			}

			size := dirSize(path)
			recent := isRecent(path, 7)
			item := model.CandidateItem{
				ID:           "purge-" + sanitizeID(path),
				RuleID:       rule.ID,
				Path:         path,
				SizeBytes:    size,
				LastModified: time.Now().UTC(),
				Category:     rule.Category,
				Risk:         rule.Risk,
				Selected:     !recent,
				RequiresRoot: rule.RequiresRoot,
				Result:       "planned",
			}

			if err := safety.ValidatePath(path, []string{home}, app.Whitelist); err != nil {
				item.Selected = false
				item.Result = "skipped"
				errCount++
			}

			if item.Selected {
				selected++
				estimate += size
			}

			items = append(items, item)
			return filepath.SkipDir
		})
	}

	if !app.Options.DryRun {
		if !app.Options.Yes {
			return model.CommandResult{}, errors.New("confirmation required for destructive action: use --yes or --dry-run")
		}
		for i := range items {
			if !items[i].Selected {
				continue
			}

			err := safety.SafeDelete(items[i].Path, []string{home}, app.Whitelist, false)
			if err != nil {
				items[i].Result = "error"
				errCount++
			} else {
				items[i].Result = "deleted"
			}

			if err := app.Logger.Log(ctx, model.OperationLogEntry{
				Timestamp: time.Now().UTC(),
				PlanID:    "plan-purge",
				Command:   "purge",
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
		Command:       "purge",
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

func defaultPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{"Projects", "Code", "dev", "src", "GitHub"}
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		p := filepath.Join(home, c)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = append(out, home)
	}
	return out
}

func isRecent(path string, days int) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(st.ModTime()) < time.Duration(days)*24*time.Hour
}

func sanitizeID(path string) string {
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.ReplaceAll(path, " ", "-")
	if len(path) > 64 {
		path = path[len(path)-64:]
	}
	return path
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
