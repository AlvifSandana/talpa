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
	_, err := NewService().Run(context.Background(), app, Options{})
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
	_, err := NewService().Run(context.Background(), app, Options{})
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestRunSystemCleanNonRootMarksSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldEUID := getEUID
	getEUID = func() int { return 1000 }
	defer func() { getEUID = oldEUID }()

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{System: true})
	if err != nil {
		t.Fatal(err)
	}

	foundSystem := false
	for _, it := range res.Items {
		if it.RequiresRoot {
			foundSystem = true
			if it.Selected {
				t.Fatalf("expected system-level item to be unselected when non-root: %s", it.Path)
			}
			if it.Result != "skipped" {
				t.Fatalf("expected system-level item skipped when non-root, got %s", it.Result)
			}
		}
	}
	if !foundSystem {
		t.Fatalf("expected at least one system-level item in system mode")
	}
}

func TestDeleteCleanTargetSystemPathDeletesChildrenOnly(t *testing.T) {
	oldReadDir := cleanReadDir
	oldDelete := cleanSafeDelete
	defer func() {
		cleanReadDir = oldReadDir
		cleanSafeDelete = oldDelete
	}()

	cleanReadDir = func(name string) ([]os.DirEntry, error) {
		if name != "/tmp" {
			return nil, os.ErrNotExist
		}
		return []os.DirEntry{
			fakeDirEntry{name: "a.tmp"},
			fakeDirEntry{name: "b.tmp"},
		}, nil
	}
	called := []string{}
	cleanSafeDelete = func(path string, allowedRoots []string, whitelist []string, dryRun bool) error {
		called = append(called, path)
		return nil
	}

	if err := deleteCleanTarget("/tmp", []string{"/tmp"}, []string{"/tmp"}, false); err != nil {
		t.Fatal(err)
	}
	if len(called) != 2 {
		t.Fatalf("expected 2 child deletions, got %d", len(called))
	}
	for _, c := range called {
		if c == "/tmp" {
			t.Fatalf("expected not to delete system root directory directly")
		}
	}
}

type fakeDirEntry struct{ name string }

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return false }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, os.ErrNotExist }
