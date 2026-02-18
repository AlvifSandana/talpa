package analyze

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/domain/safety"
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
	Action         string
	TrashDir       string
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
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return model.CommandResult{}, err
	}

	items, err := filesystem.Scan(root, filesystem.ScanOptions{
		MaxDepth:       opts.Depth,
		Excludes:       []string{"/proc", "/sys", "/dev", "/run"},
		SkipMountpoint: opts.Action != "inspect",
		SkipNetworkFS:  true,
		Context:        ctx,
	})
	if err != nil {
		return model.CommandResult{}, err
	}

	if !app.Options.DryRun && opts.Action == "delete" {
		if err := common.RequireHighRiskConfirmationOrDryRun(app.Options, "analyze delete"); err != nil {
			return model.CommandResult{}, err
		}
	} else if !app.Options.DryRun && opts.Action == "trash" {
		if err := common.RequireConfirmationOrDryRun(app.Options, "analyze trash"); err != nil {
			return model.CommandResult{}, err
		}
		if err := ensureTrashActionSupported(); err != nil {
			return model.CommandResult{}, err
		}
	}

	items = filterItems(items, opts)
	sortItems(items, opts.SortBy)
	if opts.Limit > 0 && len(items) > opts.Limit {
		items = items[:opts.Limit]
	}

	out := make([]model.CandidateItem, 0, len(items))
	var estimate int64
	candidates := 0
	errCount := 0
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
		if opts.Action == "delete" {
			if candidate {
				if app.Options.DryRun {
					result = "planned"
				} else {
					result = "deleted"
					if err := safety.SafeDeleteWithIdentity(it.Path, []string{rootAbs}, app.Whitelist, false, it.Device, it.Inode); err != nil {
						if os.IsNotExist(err) {
							result = "skipped"
						} else {
							result = "error"
							errCount++
						}
					}
				}
			} else {
				result = "skipped"
			}
		} else if opts.Action == "trash" {
			if candidate {
				if app.Options.DryRun {
					result = "planned"
				} else {
					result = "trashed"
					if err := moveToTrash(it.Path, opts.TrashDir, []string{rootAbs}, app.Whitelist, it.Device, it.Inode); err != nil {
						if os.IsNotExist(err) {
							result = "skipped"
						} else {
							result = "error"
							errCount++
						}
					}
				}
			} else {
				result = "skipped"
			}
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
			Errors:              errCount,
		},
		Items: out,
	}, nil
}

func moveToTrash(path, trashDir string, allowedRoots []string, whitelist []string, expectedDev uint64, expectedIno uint64) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("empty path")
	}
	srcAbs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	if err := safety.ValidatePath(srcAbs, allowedRoots, whitelist); err != nil {
		return err
	}
	if strings.TrimSpace(trashDir) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		trashDir = filepath.Join(home, ".local", "share", "Trash", "files")
	}
	trashAbs, err := filepath.Abs(filepath.Clean(trashDir))
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	homeAbs, err := filepath.Abs(filepath.Clean(home))
	if err != nil {
		return err
	}
	if !hasPathPrefix(trashAbs, homeAbs) {
		return errors.New("PATH_BLOCKED: trash dir must be inside user home")
	}
	if err := safety.ValidatePath(trashAbs, []string{homeAbs}, nil); err != nil {
		return err
	}
	if err := prepareTrashDir(trashAbs); err != nil {
		return err
	}
	return secureMoveToTrash(srcAbs, trashAbs, allowedRoots, whitelist, expectedDev, expectedIno)
}

func isAllowedTrashSource(path string, roots []string, whitelist []string) bool {
	resolvedPath := resolvePath(path)
	if matchesWhitelist(path, whitelist) || matchesWhitelist(resolvedPath, whitelist) {
		return true
	}
	if len(roots) == 0 {
		return true
	}
	for _, r := range roots {
		abs, err := filepath.Abs(filepath.Clean(r))
		if err != nil {
			continue
		}
		resolvedRoot := resolvePath(abs)
		if hasPathPrefix(path, abs) || hasPathPrefix(resolvedPath, resolvedRoot) {
			return true
		}
	}
	return false
}

func resolvePath(path string) string {
	r, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	abs, err := filepath.Abs(filepath.Clean(r))
	if err != nil {
		return r
	}
	return abs
}

func hasPathPrefix(path, prefix string) bool {
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+string(filepath.Separator))
}

func matchesWhitelist(path string, whitelist []string) bool {
	for _, w := range whitelist {
		if strings.ContainsAny(w, "*?[") {
			if ok, _ := filepath.Match(w, path); ok {
				return true
			}
		}
		if path == w || strings.HasPrefix(path, w+string(filepath.Separator)) {
			return true
		}
	}
	return false
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
