package clean

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/domain/safety"
)

type Service struct{}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext) (model.CommandResult, error) {
	start := time.Now()
	home, err := os.UserHomeDir()
	if err != nil {
		return model.CommandResult{}, err
	}

	candidates := []string{
		filepath.Join(home, ".cache"),
		filepath.Join(home, ".local", "share", "Trash"),
		filepath.Join(home, ".cache", "thumbnails"),
	}

	items := make([]model.CandidateItem, 0, len(candidates))
	selected := 0
	var estimate int64
	errCount := 0

	for i, p := range candidates {
		size := dirSize(p)
		item := model.CandidateItem{
			ID:           "clean-" + strconv.Itoa(i+1),
			RuleID:       "clean.default",
			Path:         p,
			SizeBytes:    size,
			LastModified: time.Now().UTC(),
			Category:     "cache",
			Risk:         model.RiskLow,
			Selected:     true,
			RequiresRoot: false,
			Result:       "planned",
		}

		if err := safety.ValidatePath(p, []string{home}, app.Whitelist); err != nil {
			item.Selected = false
			item.Result = "skipped"
			errCount++
		} else {
			selected++
			estimate += size
		}

		items = append(items, item)
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
