package uninstall

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
)

type Service struct{}

type Options struct {
	Apply bool
}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, opts Options) (model.CommandResult, error) {
	_ = ctx
	home, err := os.UserHomeDir()
	if err != nil {
		return model.CommandResult{}, err
	}

	items := []model.CandidateItem{
		newPlanItem("uninstall-1", "uninstall.binary", filepath.Join(home, ".local", "bin", "talpa"), "app_binary", model.RiskHigh),
		newPlanItem("uninstall-2", "uninstall.config", filepath.Join(home, ".config", "talpa"), "config", model.RiskMedium),
		newPlanItem("uninstall-3", "uninstall.cache", filepath.Join(home, ".cache", "talpa"), "cache", model.RiskLow),
	}

	if opts.Apply && !app.Options.DryRun {
		if err := common.RequireConfirmationOrDryRun(app.Options, "uninstall"); err != nil {
			return model.CommandResult{}, err
		}
		for i := range items {
			if err := common.ValidateSystemScopePath(items[i].Path, app.Whitelist); err != nil {
				items[i].Result = "skipped"
				continue
			}
			items[i].Result = "pending_implementation"
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
		},
		Items: items,
	}, nil
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
