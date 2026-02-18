//go:build unix
// +build unix

package analyze

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestMoveToTrashFallsBackOnEXDEV(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	src := filepath.Join(root, "cache")
	mustMkdir(t, src)
	mustWrite(t, filepath.Join(src, "a.tmp"), 16)
	trash := filepath.Join(root, "trash")

	original := renameAt
	renameAt = func(oldfd int, oldpath string, newfd int, newpath string) error {
		return syscall.EXDEV
	}
	t.Cleanup(func() { renameAt = original })

	err := moveToTrash(src, trash, []string{root}, nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source removed after fallback move")
	}
	entries, err := os.ReadDir(trash)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected moved entry in trash directory")
	}
}

func TestMoveToTrashPropagatesRenameError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	src := filepath.Join(root, "cache")
	mustMkdir(t, src)
	trash := filepath.Join(root, "trash")

	original := renameAt
	renameAt = func(oldfd int, oldpath string, newfd int, newpath string) error { return errors.New("boom") }
	t.Cleanup(func() { renameAt = original })

	err := moveToTrash(src, trash, []string{root}, nil, 0, 0)
	if err == nil {
		t.Fatalf("expected rename error")
	}
}
