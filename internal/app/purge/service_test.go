package purge

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	_, err := NewService().Run(context.Background(), app, []string{filepath.Join(home, "Projects")}, Options{})
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
	_, err := NewService().Run(context.Background(), app, []string{filepath.Join(home, "Projects")}, Options{})
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestRunHonorsDepthLimit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	deepArtifact := filepath.Join(home, "Projects", "a", "b", "c", "node_modules")
	if err := os.MkdirAll(deepArtifact, 0o755); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, []string{filepath.Join(home, "Projects")}, Options{MaxDepth: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("expected no items due to depth limit, got %d", len(res.Items))
	}
}

func TestRunRespectsRecentDaysSelection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projects := filepath.Join(home, "Projects")
	oldArtifact := filepath.Join(projects, "old", "node_modules")
	recentArtifact := filepath.Join(projects, "recent", "node_modules")
	if err := os.MkdirAll(oldArtifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(recentArtifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldArtifact, "a.js"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recentArtifact, "b.js"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(oldArtifact, old, old); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, []string{projects}, Options{RecentDays: 7})
	if err != nil {
		t.Fatal(err)
	}

	selectedByPath := map[string]bool{}
	for _, item := range res.Items {
		selectedByPath[item.Path] = item.Selected
	}
	if !selectedByPath[oldArtifact] {
		t.Fatalf("expected old artifact to be selected")
	}
	if selectedByPath[recentArtifact] {
		t.Fatalf("expected recent artifact to be skipped by default")
	}
}
