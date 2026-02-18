package remove

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
)

type Service struct{}

var (
	osExecutable = os.Executable
	osRemove     = os.Remove
	osStat       = os.Stat
)

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext) (model.CommandResult, error) {
	exe, err := osExecutable()
	if err != nil {
		return model.CommandResult{}, err
	}
	target := exe
	errCount := 0
	item := model.CandidateItem{
		ID:           "remove-1",
		RuleID:       "remove.binary",
		Path:         target,
		Category:     "self_remove",
		Risk:         model.RiskHigh,
		Selected:     true,
		RequiresRoot: isSystemPath(target),
		Result:       "planned",
	}

	if !app.Options.DryRun {
		if err := common.RequireHighRiskConfirmationOrDryRun(app.Options, "remove"); err != nil {
			return model.CommandResult{}, err
		}
		if err := common.ValidateSystemScopePath(target, app.Whitelist); err != nil {
			item.Result = "skipped"
			errCount++
		} else if err := osRemove(target); err != nil {
			item.Result = "error"
			errCount++
		} else {
			item.Result = "deleted"
		}
		if err := app.Logger.Log(ctx, model.OperationLogEntry{
			Timestamp: time.Now().UTC(),
			PlanID:    "plan-remove",
			Command:   "remove",
			Action:    "delete",
			Path:      target,
			RuleID:    item.RuleID,
			Category:  item.Category,
			Risk:      string(item.Risk),
			Result:    item.Result,
			DryRun:    false,
		}); err != nil {
			errCount++
		}
	}

	item.LastModified = time.Now().UTC()
	if st, err := osStat(target); err == nil {
		item.SizeBytes = st.Size()
	}
	if app.Options.DryRun {
		item.Result = "planned"
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "remove",
		Timestamp:     time.Now().UTC(),
		DurationMS:    0,
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:    1,
			ItemsSelected: 1,
			Errors:        errCount,
		},
		Items: []model.CandidateItem{item},
	}, nil
}

func isSystemPath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	return strings.HasPrefix(abs, "/usr/") || strings.HasPrefix(abs, "/bin/") || strings.HasPrefix(abs, "/sbin/")
}
