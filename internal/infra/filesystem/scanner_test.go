package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanHonorsDepth(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "root.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deep, "b.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	items, err := Scan(root, ScanOptions{MaxDepth: 2, Concurrency: 2, Timeout: time.Second})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	paths := make(map[string]struct{}, len(items))
	for _, it := range items {
		paths[it.Path] = struct{}{}
	}

	mustHave := []string{filepath.Join(root, "root.txt"), filepath.Join(root, "a", "a.txt")}
	for _, p := range mustHave {
		if _, ok := paths[filepath.Clean(p)]; !ok {
			t.Fatalf("expected path %s", p)
		}
	}

	if _, ok := paths[filepath.Clean(filepath.Join(deep, "b.txt"))]; ok {
		t.Fatalf("unexpected deep path included")
	}
}

func TestScanHonorsExcludes(t *testing.T) {
	root := t.TempDir()
	keepDir := filepath.Join(root, "keep")
	skipDir := filepath.Join(root, "skip")
	if err := os.MkdirAll(keepDir, 0o755); err != nil {
		t.Fatalf("mkdir keep: %v", err)
	}
	if err := os.MkdirAll(skipDir, 0o755); err != nil {
		t.Fatalf("mkdir skip: %v", err)
	}
	if err := os.WriteFile(filepath.Join(keepDir, "ok.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write keep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skipDir, "no.txt"), []byte("no"), 0o644); err != nil {
		t.Fatalf("write skip: %v", err)
	}

	items, err := Scan(root, ScanOptions{Excludes: []string{skipDir}, Concurrency: 2, Timeout: time.Second})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	for _, it := range items {
		if filepath.Dir(it.Path) == filepath.Clean(skipDir) {
			t.Fatalf("excluded directory item found: %s", it.Path)
		}
	}
}

func TestParseMountInfo(t *testing.T) {
	raw := "123 456 0:45 / / rw,relatime - ext4 /dev/sda1 rw\n" +
		"124 456 0:46 / /mnt/nfs rw,relatime - nfs server:/export rw\n"
	m := parseMountInfo(raw)
	if got := m["/"]; got != "ext4" {
		t.Fatalf("expected ext4 root, got %q", got)
	}
	if got := m["/mnt/nfs"]; got != "nfs" {
		t.Fatalf("expected nfs mount, got %q", got)
	}
}

func TestIsNetworkFS(t *testing.T) {
	if !isNetworkFS("nfs4") {
		t.Fatalf("expected nfs4 as network fs")
	}
	if isNetworkFS("ext4") {
		t.Fatalf("did not expect ext4 as network fs")
	}
}

func TestShouldSkipMountSemantics(t *testing.T) {
	root := "/root"
	mounts := map[string]string{
		"/root/mnt/nfs": "nfs",
		"/root/mnt/ext": "ext4",
	}

	if !shouldSkipMount("/root/mnt/nfs", root, ScanOptions{SkipNetworkFS: true}, mounts) {
		t.Fatalf("expected network fs mount to be skipped")
	}
	if shouldSkipMount("/root/mnt/ext", root, ScanOptions{SkipNetworkFS: true}, mounts) {
		t.Fatalf("did not expect local mount to be skipped when only SkipNetworkFS enabled")
	}
	if !shouldSkipMount("/root/mnt/ext", root, ScanOptions{SkipMountpoint: true}, mounts) {
		t.Fatalf("expected mountpoint to be skipped when SkipMountpoint enabled")
	}
}

func TestWithinRoot(t *testing.T) {
	if !withinRoot("/a/b", "/a") {
		t.Fatalf("expected nested path within root")
	}
	if withinRoot("/ab", "/a") {
		t.Fatalf("unexpected prefix-only match")
	}
}

func TestScanFailsWhenMountMetadataUnavailableAndSkippingRequested(t *testing.T) {
	root := t.TempDir()
	original := readMountPointsFunc
	readMountPointsFunc = func() (map[string]string, error) {
		return nil, errors.New("mountinfo unavailable")
	}
	t.Cleanup(func() { readMountPointsFunc = original })

	_, err := Scan(root, ScanOptions{SkipMountpoint: true, Timeout: time.Second})
	if err == nil {
		t.Fatalf("expected scan to fail when mount metadata unavailable and skip mountpoint requested")
	}
}
