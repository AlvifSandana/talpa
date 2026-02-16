package common

import (
	"context"
	"time"

	"talpa/internal/domain/model"
	"talpa/internal/infra/logging"
)

func LogApplySkip(ctx context.Context, logger logging.Logger, planID, command string, item model.CandidateItem) error {
	entry := model.OperationLogEntry{
		Timestamp: time.Now().UTC(),
		PlanID:    planID,
		Command:   command,
		Action:    "skip",
		Path:      item.Path,
		RuleID:    item.RuleID,
		Category:  item.Category,
		Risk:      string(item.Risk),
		Result:    item.Result,
		DryRun:    false,
	}
	if entry.Result == "" {
		entry.Result = "skipped"
	}
	return logger.Log(ctx, entry)
}
