package safety

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var blockedPaths = []string{
	"/",
	"/boot",
	"/bin",
	"/sbin",
	"/lib",
	"/lib64",
	"/usr",
	"/etc",
	"/proc",
	"/sys",
	"/dev",
	"/run",
	"/var",
}

func ValidatePath(path string, allowedRoots []string, whitelist []string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("PATH_INVALID: empty path")
	}
	if strings.ContainsRune(path, rune(0)) {
		return errors.New("PATH_INVALID: null byte")
	}
	for _, r := range path {
		if r < 32 {
			return errors.New("PATH_INVALID: control character")
		}
	}
	if strings.Contains(path, "/../") {
		return errors.New("PATH_INVALID: traversal")
	}

	clean := filepath.Clean(path)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return fmt.Errorf("PATH_INVALID: %w", err)
	}

	if isBlocked(abs) && !isWhitelisted(abs, whitelist) {
		return fmt.Errorf("PATH_BLOCKED: %s", abs)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		if isBlocked(resolved) && !isWhitelisted(resolved, whitelist) {
			return fmt.Errorf("SYMLINK_ESCAPE: %s", resolved)
		}
		if !inAllowedRoots(resolved, allowedRoots) {
			return fmt.Errorf("SYMLINK_ESCAPE: %s", resolved)
		}
	}

	if !inAllowedRoots(abs, allowedRoots) && !isWhitelisted(abs, whitelist) {
		return fmt.Errorf("PATH_BLOCKED: outside allowed roots %s", abs)
	}

	return nil
}

func SafeDelete(path string, allowedRoots []string, whitelist []string, dryRun bool) error {
	if err := ValidatePath(path, allowedRoots, whitelist); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return os.RemoveAll(path)
}

func isBlocked(path string) bool {
	for _, p := range blockedPaths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func inAllowedRoots(path string, roots []string) bool {
	if len(roots) == 0 {
		return true
	}
	for _, r := range roots {
		abs, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		if path == abs || strings.HasPrefix(path, abs+"/") {
			return true
		}
	}
	return false
}

func isWhitelisted(path string, whitelist []string) bool {
	for _, w := range whitelist {
		if path == w || strings.HasPrefix(path, w+"/") {
			return true
		}
	}
	return false
}
