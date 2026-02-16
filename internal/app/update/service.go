package update

import (
	"context"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
)

type Service struct{}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext) (model.CommandResult, error) {
	_ = ctx
	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "update",
		Timestamp:     time.Now().UTC(),
		DurationMS:    0,
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:    1,
			ItemsSelected: 1,
		},
		Items: []model.CandidateItem{
			{
				ID:           "update-1",
				RuleID:       "update.binary",
				Path:         "/usr/local/bin/talpa",
				Category:     "self_update",
				Risk:         model.RiskMedium,
				Selected:     true,
				RequiresRoot: true,
				Result:       "planned",
			},
		},
	}, nil
}
