package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

	items := []model.CandidateItem{
		newPlanItem("installer-1", "installer.download", filepath.Join(home, "Downloads", "talpa-installer.sh"), "installer_artifact", model.RiskLow),
		newPlanItem("installer-2", "installer.download.sig", filepath.Join(home, "Downloads", "talpa-installer.sh.sha256"), "installer_artifact", model.RiskLow),
		newPlanItem("installer-3", "installer.tmp", filepath.Join("/tmp", "talpa-installer"), "installer_artifact", model.RiskLow),
	}
	for i := range items {
		if _, err := osStat(items[i].Path); errors.Is(err, os.ErrNotExist) {
			items[i].Selected = false
			items[i].Result = "skipped"
		}
	}

	selected := 0
	for _, item := range items {
		if item.Selected {
			selected++
		}
	}

	errCount := 0
	if opts.Apply {
		if err := common.RequireConfirmationOrDryRun(app.Options, "installer cleanup"); err != nil {
			return model.CommandResult{}, err
		}
		if !app.Options.DryRun {
			for i := range items {
				if !items[i].Selected {
					if err := common.LogApplySkip(ctx, app.Logger, "plan-installer", "installer", items[i]); err != nil {
						errCount++
					}
					continue
				}
				entry := model.OperationLogEntry{
					Timestamp: time.Now().UTC(),
					PlanID:    "plan-installer",
					Command:   "installer",
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
		Command:       "installer",
		Timestamp:     time.Now().UTC(),
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:    len(items),
			ItemsSelected: selected,
			Errors:        errCount,
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
