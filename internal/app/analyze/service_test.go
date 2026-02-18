package analyze

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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

func TestRunDeleteActionRequiresHighRiskDoubleConfirm(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "project", "cache"))
	mustWrite(t, filepath.Join(root, "project", "cache", "a.tmp"), 16)

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, root, Options{Depth: 4, Limit: 10, SortBy: "size", Action: "delete"})
	if err == nil {
		t.Fatalf("expected high-risk double confirmation error")
	}
}

func TestRunDeleteActionUsesSafetyGate(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "project", "cache")
	mustMkdir(t, cacheDir)
	target := filepath.Join(cacheDir, "a.tmp")
	mustWrite(t, target, 16)

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, root, Options{Depth: 4, Limit: 10, SortBy: "size", Action: "delete"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) == 0 {
		t.Fatalf("expected analyze items")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected target to be deleted")
	}
}

func TestRunActionDryRunReportsPlannedForMutatingActions(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "project", "cache"))
	mustWrite(t, filepath.Join(root, "project", "cache", "a.tmp"), 16)

	for _, action := range []string{"delete", "trash"} {
		t.Run(action, func(t *testing.T) {
			app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
			res, err := NewService().Run(context.Background(), app, root, Options{Depth: 4, Limit: 10, SortBy: "size", Action: action})
			if err != nil {
				t.Fatal(err)
			}
			foundPlanned := false
			for _, it := range res.Items {
				if it.Selected {
					if it.Result != "planned" {
						t.Fatalf("expected selected item result planned for dry-run, got %q", it.Result)
					}
					foundPlanned = true
				}
			}
			if !foundPlanned {
				t.Fatalf("expected at least one selected item")
			}
		})
	}
}

func TestRunTrashActionSkipsMissingDescendantAfterParentMoved(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "project", "cache")
	child := filepath.Join(parent, "nested")
	mustMkdir(t, child)
	mustWrite(t, filepath.Join(child, "a.tmp"), 16)

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, root, Options{Depth: 6, Limit: 50, SortBy: "size", Action: "trash"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Errors != 0 {
		t.Fatalf("expected no errors when descendant is already moved with parent, got %d", res.Summary.Errors)
	}
}

func TestMoveToTrashRejectsSymlinkTrashDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions and behavior differ on windows")
	}
	root := t.TempDir()
	src := filepath.Join(root, "cache")
	realTrash := filepath.Join(root, "real-trash")
	symlinkTrash := filepath.Join(root, "link-trash")
	mustMkdir(t, src)
	mustMkdir(t, realTrash)
	if err := os.Symlink(realTrash, symlinkTrash); err != nil {
		t.Skipf("unable to create symlink: %v", err)
	}

	err := moveToTrash(src, symlinkTrash, []string{root}, nil, 0, 0)
	if err == nil {
		t.Fatalf("expected symlink trash dir to be rejected")
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
