package installer

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
		newPlanItem("installer-1", "installer.download", filepath.Join(home, "Downloads", "talpa-installer.sh"), "installer_artifact", model.RiskLow),
		newPlanItem("installer-2", "installer.download.sig", filepath.Join(home, "Downloads", "talpa-installer.sh.sha256"), "installer_artifact", model.RiskLow),
		newPlanItem("installer-3", "installer.tmp", filepath.Join("/tmp", "talpa-installer"), "installer_artifact", model.RiskLow),
	}

	if opts.Apply && !app.Options.DryRun {
		if err := common.RequireConfirmationOrDryRun(app.Options, "installer cleanup"); err != nil {
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
		Command:       "installer",
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
