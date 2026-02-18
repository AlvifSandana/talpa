package safety

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePathRejectsBlockedPath(t *testing.T) {
	err := ValidatePath("/etc", nil, nil)
	if err == nil {
		t.Fatal("expected blocked path error")
	}
}

func TestValidatePathAllowsPathWithinRoot(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "cache")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}

	err := ValidatePath(p, []string{root}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidatePathRejectsOutsideAllowedRoot(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()

	err := ValidatePath(other, []string{root}, nil)
	if err == nil {
		t.Fatal("expected outside root error")
	}
}

func TestValidatePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	err := ValidatePath(link, []string{root}, nil)
	if err == nil {
		t.Fatal("expected symlink escape error")
	}
}

func TestValidatePathAllowsWhitelistedGlob(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "cache", "a")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	err := ValidatePath(p, nil, []string{filepath.Join(root, "cache", "*")})
	if err != nil {
		t.Fatalf("expected glob-whitelisted path to pass, got %v", err)
	}
}
