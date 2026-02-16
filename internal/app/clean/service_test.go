package clean

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"talpa/internal/app/common"
	"talpa/internal/infra/logging"
)

func TestRunDryRunDoesNotDelete(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cache := filepath.Join(home, ".cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "x.tmp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(cache); err != nil {
		t.Fatalf("expected cache dir to exist in dry-run: %v", err)
	}
}

func TestRunRequiresYesForDestructive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cache := filepath.Join(home, ".cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: false}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app)
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}
