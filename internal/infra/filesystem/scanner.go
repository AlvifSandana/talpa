package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ScanItem struct {
	Path         string
	SizeBytes    int64
	LastModified time.Time
}

type ScanOptions struct {
	MaxDepth int
	Excludes []string
}

func Scan(root string, opts ScanOptions) ([]ScanItem, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	items := make([]ScanItem, 0, 128)
	err = filepath.Walk(rootAbs, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if shouldSkip(path, opts.Excludes) {
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if opts.MaxDepth > 0 && depth(rootAbs, path) > opts.MaxDepth {
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info == nil || info.IsDir() {
			return nil
		}

		items = append(items, ScanItem{
			Path:         path,
			SizeBytes:    info.Size(),
			LastModified: info.ModTime().UTC(),
		})
		return nil
	})

	return items, err
}

func shouldSkip(path string, excludes []string) bool {
	for _, ex := range excludes {
		if ex == "" {
			continue
		}
		if path == ex || strings.HasPrefix(path, ex+"/") {
			return true
		}
	}
	return false
}

func depth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}
