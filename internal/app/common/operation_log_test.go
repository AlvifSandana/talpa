package common

import (
	"context"
	"errors"
	"testing"

	"talpa/internal/domain/model"
)

type captureCommonLogger struct {
	entries []model.OperationLogEntry
	err     error
}

func (c *captureCommonLogger) Log(_ context.Context, entry model.OperationLogEntry) error {
	c.entries = append(c.entries, entry)
	return c.err
}

func TestLogApplySkipDefaultsSkippedResult(t *testing.T) {
	logger := &captureCommonLogger{}
	item := model.CandidateItem{
		Path:     "/tmp/talpa-installer",
		RuleID:   "installer.tmp",
		Category: "installer_artifact",
		Risk:     model.RiskLow,
	}

	err := LogApplySkip(context.Background(), logger, "plan-installer", "installer", item)
	if err != nil {
		t.Fatal(err)
	}
	if len(logger.entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(logger.entries))
	}
	e := logger.entries[0]
	if e.Action != "skip" {
		t.Fatalf("expected skip action, got %q", e.Action)
	}
	if e.Result != "skipped" {
		t.Fatalf("expected skipped result default, got %q", e.Result)
	}
	if e.PlanID != "plan-installer" || e.Command != "installer" {
		t.Fatalf("unexpected plan/command metadata: %+v", e)
	}
}

func TestLogApplySkipPreservesExistingResult(t *testing.T) {
	logger := &captureCommonLogger{}
	item := model.CandidateItem{
		Path:     "/tmp/talpa",
		RuleID:   "uninstall.binary",
		Category: "app_binary",
		Risk:     model.RiskHigh,
		Result:   "already-skipped",
	}

	err := LogApplySkip(context.Background(), logger, "plan-uninstall", "uninstall", item)
	if err != nil {
		t.Fatal(err)
	}
	if logger.entries[0].Result != "already-skipped" {
		t.Fatalf("expected existing result to be preserved, got %q", logger.entries[0].Result)
	}
}

func TestLogApplySkipPropagatesLoggerError(t *testing.T) {
	logger := &captureCommonLogger{err: errors.New("write failed")}
	err := LogApplySkip(context.Background(), logger, "plan-uninstall", "uninstall", model.CandidateItem{})
	if err == nil {
		t.Fatal("expected logger error to be returned")
	}
}
