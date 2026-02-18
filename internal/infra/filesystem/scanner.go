package filesystem

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type ScanItem struct {
	Path         string
	SizeBytes    int64
	LastModified time.Time
}

type ScanOptions struct {
	MaxDepth       int
	Excludes       []string
	Concurrency    int
	Timeout        time.Duration
	SkipNetworkFS  bool
	SkipMountpoint bool
	Context        context.Context
}

func Scan(root string, opts ScanOptions) ([]ScanItem, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = runtime.NumCPU()
		if opts.Concurrency < 2 {
			opts.Concurrency = 2
		}
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootAbs = filepath.Clean(rootAbs)

	excludes := normalizePaths(opts.Excludes)
	mounts := readMountPoints()
	rootResolved := rootAbs
	if resolved, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootResolved = filepath.Clean(resolved)
	}

	baseCtx := opts.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithTimeout(baseCtx, opts.Timeout)
	defer cancel()

	var (
		itemsMu sync.Mutex
		items   = make([]ScanItem, 0, 256)
	)

	type dirQueue struct {
		mu      sync.Mutex
		cond    *sync.Cond
		dirs    []string
		pending int
	}

	q := &dirQueue{dirs: []string{rootAbs}, pending: 1}
	q.cond = sync.NewCond(&q.mu)

	push := func(path string) {
		q.mu.Lock()
		q.dirs = append(q.dirs, path)
		q.pending++
		q.cond.Signal()
		q.mu.Unlock()
	}

	done := func() {
		q.mu.Lock()
		q.pending--
		if q.pending <= 0 {
			q.pending = 0
			q.cond.Broadcast()
		}
		q.mu.Unlock()
	}

	pop := func() (string, bool) {
		q.mu.Lock()
		defer q.mu.Unlock()

		for len(q.dirs) == 0 && q.pending > 0 {
			if ctx.Err() != nil {
				return "", false
			}
			q.cond.Wait()
		}

		if len(q.dirs) == 0 {
			return "", false
		}

		d := q.dirs[0]
		q.dirs = q.dirs[1:]
		return d, true
	}

	var workers sync.WaitGroup
	for i := 0; i < opts.Concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()

			for {
				dir, ok := pop()
				if !ok {
					return
				}

				func() {
					defer done()

					select {
					case <-ctx.Done():
						return
					default:
					}

					resolvedDir := dir
					if r, err := filepath.EvalSymlinks(dir); err == nil {
						resolvedDir = filepath.Clean(r)
					}
					if !withinRoot(resolvedDir, rootResolved) {
						return
					}

					entries, err := os.ReadDir(dir)
					if err != nil {
						return
					}

					for _, entry := range entries {
						select {
						case <-ctx.Done():
							return
						default:
						}

						path := filepath.Clean(filepath.Join(dir, entry.Name()))

						if shouldSkip(path, excludes) {
							continue
						}

						if opts.MaxDepth > 0 && depth(rootAbs, path) > opts.MaxDepth {
							continue
						}

						if shouldSkipMount(path, rootAbs, opts, mounts) {
							continue
						}

						if entry.Type()&os.ModeSymlink != 0 {
							continue
						}

						info, err := entry.Info()
						if err != nil || info == nil {
							continue
						}

						if info.IsDir() {
							push(path)
							continue
						}

						itemsMu.Lock()
						items = append(items, ScanItem{
							Path:         path,
							SizeBytes:    info.Size(),
							LastModified: info.ModTime().UTC(),
						})
						itemsMu.Unlock()
					}
				}()
			}
		}()
	}

	go func() {
		<-ctx.Done()
		q.mu.Lock()
		q.cond.Broadcast()
		q.mu.Unlock()
	}()

	workers.Wait()

	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return items, context.DeadlineExceeded
	}

	return items, nil
}

func shouldSkipMount(path, root string, opts ScanOptions, mounts map[string]string) bool {
	if path == root {
		return false
	}
	fsType, ok := mounts[path]
	if !ok {
		return false
	}
	if opts.SkipNetworkFS && isNetworkFS(fsType) {
		return true
	}
	if opts.SkipMountpoint {
		return true
	}
	return false
}

func shouldSkip(path string, excludes []string) bool {
	for _, ex := range excludes {
		if ex == "" {
			continue
		}
		if path == ex || strings.HasPrefix(path, ex+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func normalizePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		out = append(out, filepath.Clean(abs))
	}
	return out
}

func readMountPoints() map[string]string {
	b, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return map[string]string{}
	}
	return parseMountInfo(string(b))
}

func parseMountInfo(raw string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			continue
		}

		left := strings.Fields(parts[0])
		right := strings.Fields(parts[1])
		if len(left) < 5 || len(right) < 1 {
			continue
		}

		mountPoint := unescapeMountInfoField(left[4])
		mountPoint = filepath.Clean(mountPoint)
		if mountPoint == "" {
			continue
		}

		out[mountPoint] = strings.ToLower(strings.TrimSpace(right[0]))
	}
	return out
}

func unescapeMountInfoField(s string) string {
	repl := strings.NewReplacer(
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, "\\",
	)
	return repl.Replace(s)
}

func isNetworkFS(fsType string) bool {
	networkFS := map[string]struct{}{
		"nfs":        {},
		"nfs4":       {},
		"cifs":       {},
		"smbfs":      {},
		"sshfs":      {},
		"fuse.sshfs": {},
		"9p":         {},
		"davfs":      {},
		"ceph":       {},
		"glusterfs":  {},
	}
	_, ok := networkFS[strings.ToLower(strings.TrimSpace(fsType))]
	return ok
}

func depth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}

func withinRoot(path, root string) bool {
	if path == root {
		return true
	}
	return strings.HasPrefix(path, root+string(filepath.Separator))
}
