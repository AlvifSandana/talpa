package purge

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

	proj := filepath.Join(home, "Projects", "app", "node_modules")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "a.js"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, []string{filepath.Join(home, "Projects")})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(proj); err != nil {
		t.Fatalf("expected artifact dir to exist in dry-run: %v", err)
	}
}

func TestRunRequiresYesForDestructive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	proj := filepath.Join(home, "Projects", "app", "node_modules")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: false}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, []string{filepath.Join(home, "Projects")})
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}
