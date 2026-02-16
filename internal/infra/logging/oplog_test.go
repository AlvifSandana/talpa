package logging

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"talpa/internal/domain/model"
)

func TestOperationLoggerWritesJSONL(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	logger, err := NewOperationLogger(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}

	err = logger.Log(context.Background(), model.OperationLogEntry{
		PlanID:  "p1",
		Command: "clean",
		Action:  "delete",
		Path:    "/tmp/x",
		Result:  "success",
	})
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(configHome, "talpa", "operations.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("expected log content")
	}
}
