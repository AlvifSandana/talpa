//go:build unix
// +build unix

package analyze

import (
	"context"
	"path/filepath"
	"syscall"
	"testing"

	"talpa/internal/app/common"
	"talpa/internal/infra/logging"
)

func TestRunTrashActionEXDEVFallbackRejectsFIFO(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "project", "cache")
	mustMkdir(t, cacheDir)
	fifoPath := filepath.Join(cacheDir, "pipe")
	if err := syscall.Mkfifo(fifoPath, 0o644); err != nil {
		t.Skipf("mkfifo not available: %v", err)
	}

	original := renameAt
	renameAt = func(oldfd int, oldpath string, newfd int, newpath string) error {
		return syscall.EXDEV
	}
	t.Cleanup(func() { renameAt = original })

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, root, Options{Depth: 6, Limit: 20, SortBy: "size", Action: "trash"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Errors == 0 {
		t.Fatalf("expected at least one error for unsupported fifo in EXDEV fallback")
	}
}
