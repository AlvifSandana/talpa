package safety

import (
	"errors"
	"fmt"
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
	if hasTraversalSegment(path) {
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
	return safeDelete(path, allowedRoots, whitelist, dryRun, 0, 0, false)
}

func SafeDeleteWithIdentity(path string, allowedRoots []string, whitelist []string, dryRun bool, expectedDev uint64, expectedIno uint64) error {
	return safeDelete(path, allowedRoots, whitelist, dryRun, expectedDev, expectedIno, true)
}

func safeDelete(path string, allowedRoots []string, whitelist []string, dryRun bool, expectedDev uint64, expectedIno uint64, enforceIdentity bool) error {
	if err := ValidatePath(path, allowedRoots, whitelist); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	clean := filepath.Clean(path)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return err
	}
	if !enforceIdentity {
		return secureRemoveAll(abs, nil)
	}
	return secureRemoveAll(abs, &entryIdentity{dev: expectedDev, ino: expectedIno})
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
	pathResolved, pathResolvedErr := filepath.EvalSymlinks(path)
	for _, r := range roots {
		abs, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		if isWithinPath(path, abs) {
			return true
		}
		if rootResolved, err := filepath.EvalSymlinks(abs); err == nil && pathResolvedErr == nil {
			if isWithinPath(pathResolved, rootResolved) {
				return true
			}
		}
	}
	return false
}

func hasTraversalSegment(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, seg := range strings.Split(normalized, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

func isWhitelisted(path string, whitelist []string) bool {
	for _, w := range whitelist {
		if strings.ContainsAny(w, "*?[") {
			if ok, _ := filepath.Match(w, path); ok {
				return true
			}
		}
		if isWithinPath(path, w) {
			return true
		}
	}
	return false
}

func isWithinPath(path, root string) bool {
	pathClean := filepath.Clean(path)
	rootClean := filepath.Clean(root)
	rel, err := filepath.Rel(rootClean, pathClean)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	up := ".." + string(filepath.Separator)
	return rel != ".." && !strings.HasPrefix(rel, up)
}
