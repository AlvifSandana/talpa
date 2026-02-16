package analyze

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/infra/logging"
)

func TestRunFiltersAndOnlyCandidates(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "project", "node_modules"))
	mustMkdir(t, filepath.Join(root, "project", "docs"))
	mustWrite(t, filepath.Join(root, "project", "node_modules", "a.js"), 200)
	mustWrite(t, filepath.Join(root, "project", "docs", "readme.md"), 50)

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, root, Options{
		Depth:          6,
		Limit:          20,
		SortBy:         "size",
		MinSizeBytes:   100,
		Query:          "node_modules",
		OnlyCandidates: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("expected one candidate item, got %d", len(res.Items))
	}
	if !res.Items[0].Selected || res.Items[0].Result != "candidate" {
		t.Fatalf("expected selected candidate item")
	}
}

func TestSortByPath(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "b"))
	mustMkdir(t, filepath.Join(root, "a"))
	mustWrite(t, filepath.Join(root, "b", "x.bin"), 100)
	mustWrite(t, filepath.Join(root, "a", "y.bin"), 100)

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, root, Options{Depth: 4, Limit: 10, SortBy: "path"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) < 2 {
		t.Fatalf("expected at least two items")
	}
	if res.Items[0].Path > res.Items[1].Path {
		t.Fatalf("expected path-sorted items")
	}
}

func TestSortByMtime(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.bin")
	newPath := filepath.Join(root, "new.bin")
	mustWrite(t, oldPath, 10)
	time.Sleep(10 * time.Millisecond)
	mustWrite(t, newPath, 10)

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, root, Options{Depth: 3, Limit: 10, SortBy: "mtime"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) < 2 {
		t.Fatalf("expected at least two items")
	}
	if res.Items[0].Path != newPath {
		t.Fatalf("expected newest file first, got %s", res.Items[0].Path)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, size int) {
	t.Helper()
	b := make([]byte, size)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
