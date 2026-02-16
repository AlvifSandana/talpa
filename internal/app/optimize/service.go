package optimize

import (
	"context"
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
	items := []model.CandidateItem{
		newPlanItem("optimize-1", "optimize.package_cache", "/var/cache", "optimization", model.RiskMedium, true),
		newPlanItem("optimize-2", "optimize.journal", "/var/log/journal", "optimization", model.RiskMedium, true),
		newPlanItem("optimize-3", "optimize.trim", "/", "optimization", model.RiskHigh, true),
	}

	if opts.Apply && !app.Options.DryRun {
		if err := common.RequireConfirmationOrDryRun(app.Options, "optimize"); err != nil {
			return model.CommandResult{}, err
		}
		for i := range items {
			items[i].Result = "pending_implementation"
		}
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "optimize",
		Timestamp:     time.Now().UTC(),
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:    len(items),
			ItemsSelected: len(items),
		},
		Items: items,
	}, nil
}

func newPlanItem(id, ruleID, path, category string, risk model.RiskLevel, requiresRoot bool) model.CandidateItem {
	return model.CandidateItem{
		ID:           id,
		RuleID:       ruleID,
		Path:         path,
		Category:     category,
		Risk:         risk,
		Selected:     true,
		RequiresRoot: requiresRoot,
		LastModified: time.Now().UTC(),
		Result:       "planned",
	}
}
