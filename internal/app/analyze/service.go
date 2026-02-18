package analyze

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/infra/filesystem"
)

type Service struct{}

type Options struct {
	Depth          int
	Limit          int
	SortBy         string
	MinSizeBytes   int64
	Query          string
	OnlyCandidates bool
}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, root string, opts Options) (model.CommandResult, error) {
	_ = ctx
	start := time.Now()
	if root == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return model.CommandResult{}, err
		}
		root = h
	}

	items, err := filesystem.Scan(root, filesystem.ScanOptions{
		MaxDepth:       opts.Depth,
		Excludes:       []string{"/proc", "/sys", "/dev", "/run"},
		SkipMountpoint: false,
		SkipNetworkFS:  true,
		Context:        ctx,
	})
	if err != nil {
		return model.CommandResult{}, err
	}

	items = filterItems(items, opts)
	sortItems(items, opts.SortBy)
	if opts.Limit > 0 && len(items) > opts.Limit {
		items = items[:opts.Limit]
	}

	out := make([]model.CandidateItem, 0, len(items))
	var estimate int64
	candidates := 0
	for i, it := range items {
		candidate := isCleanupCandidate(it.Path)
		if candidate {
			estimate += it.SizeBytes
			candidates++
		}
		result := "inspect"
		if candidate {
			result = "candidate"
		}
		out = append(out, model.CandidateItem{
			ID:           "analyze-" + strconv.Itoa(i+1),
			RuleID:       "",
			Path:         it.Path,
			SizeBytes:    it.SizeBytes,
			LastModified: it.LastModified,
			Category:     "tree_node",
			Risk:         model.RiskMedium,
			Selected:     candidate,
			RequiresRoot: false,
			Result:       result,
		})
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "analyze",
		Timestamp:     time.Now().UTC(),
		DurationMS:    time.Since(start).Milliseconds(),
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:          len(out),
			ItemsSelected:       candidates,
			EstimatedFreedBytes: estimate,
			Errors:              0,
		},
		Items: out,
	}, nil
}

func filterItems(items []filesystem.ScanItem, opts Options) []filesystem.ScanItem {
	if opts.MinSizeBytes <= 0 && strings.TrimSpace(opts.Query) == "" && !opts.OnlyCandidates {
		return items
	}
	q := strings.ToLower(strings.TrimSpace(opts.Query))
	out := make([]filesystem.ScanItem, 0, len(items))
	for _, it := range items {
		if it.SizeBytes < opts.MinSizeBytes {
			continue
		}
		pathLower := strings.ToLower(it.Path)
		if q != "" && !strings.Contains(pathLower, q) {
			continue
		}
		if opts.OnlyCandidates && !isCleanupCandidate(it.Path) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func sortItems(items []filesystem.ScanItem, sortBy string) {
	switch sortBy {
	case "path":
		sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	case "mtime":
		sort.Slice(items, func(i, j int) bool { return items[i].LastModified.After(items[j].LastModified) })
	default:
		sort.Slice(items, func(i, j int) bool { return items[i].SizeBytes > items[j].SizeBytes })
	}
}

func isCleanupCandidate(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, n := range []string{"cache", "tmp", "temp", "log", "logs", "thumbnail", "thumbnails", "node_modules", "target", "dist", "build", "venv", ".venv", "__pycache__", ".tox", ".mypy_cache", "coverage", ".gradle", ".npm", ".yarn"} {
		if base == n {
			return true
		}
	}

	p := strings.ToLower(path)
	for _, seg := range []string{"/.cache/", "/cache/", "/tmp/", "/temp/", "/logs/", "/log/", "/node_modules/", "/target/", "/dist/", "/build/", "/__pycache__/", "/.gradle/", "/.npm/", "/.yarn/"} {
		if strings.Contains(p, seg) {
			return true
		}
	}
	return false
}
