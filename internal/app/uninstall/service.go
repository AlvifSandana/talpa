package uninstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/domain/safety"
)

type Service struct{}

type Options struct {
	Apply   bool
	Targets []string
}

type uninstallAdapter struct {
	Backend      string
	Executable   string
	RequiresRoot bool
	BuildCommand func(name string) []string
}

type uninstallTarget struct {
	Backend string
	Name    string
}

var (
	osUserHomeDir      = os.UserHomeDir
	osExecutable       = os.Executable
	osStat             = os.Stat
	osReadDir          = os.ReadDir
	lookPath           = exec.LookPath
	absPath            = filepath.Abs
	evalSymlinks       = filepath.EvalSymlinks
	resolveExec        = resolveTrustedExecutable
	runCmd             = runCommand
	getEUID            = os.Geteuid
	safeDelete         = safety.SafeDelete
	pathValidateSystem = common.ValidateSystemScopePath
)

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, opts Options) (model.CommandResult, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return model.CommandResult{}, err
	}
	exe, exeErr := osExecutable()
	binaryTargets := uninstallBinaryTargets(home, exe, exeErr)
	adapters := uninstallAdapters()
	availableExecutables := make(map[string]string, len(adapters))
	for _, a := range adapters {
		resolved, err := resolveExec(a.Executable)
		if err != nil {
			continue
		}
		availableExecutables[a.Backend] = resolved
	}
	targets, err := parseUninstallTargets(opts.Targets)
	if err != nil {
		return model.CommandResult{}, err
	}

	items := make([]model.CandidateItem, 0, len(binaryTargets)+2+len(targets))
	for i, target := range binaryTargets {
		id := fmt.Sprintf("uninstall-%d", i+1)
		ruleID := "uninstall.binary"
		if i > 0 {
			ruleID = "uninstall.binary.system"
		}
		items = append(items, newPlanItem(id, ruleID, target, "app_binary", model.RiskHigh))
	}
	items = append(items,
		newPlanItem("uninstall-3", "uninstall.config", filepath.Join(home, ".config", "talpa"), "config", model.RiskMedium),
		newPlanItem("uninstall-4", "uninstall.cache", filepath.Join(home, ".cache", "talpa"), "cache", model.RiskLow),
	)
	items = append(items, discoverUninstallArtifacts(home, len(items)+1)...)
	for i, t := range targets {
		adapter, ok := adapterForBackend(t.Backend, adapters)
		if !ok {
			continue
		}
		selected := true
		result := "planned"
		if _, ok := availableExecutables[t.Backend]; !ok {
			selected = false
			result = "skipped"
		}
		items = append(items, model.CandidateItem{
			ID:           fmt.Sprintf("uninstall-target-%d", i+1),
			RuleID:       fmt.Sprintf("uninstall.pkg.%s", t.Backend),
			Path:         fmt.Sprintf("%s:%s", t.Backend, t.Name),
			Category:     "package",
			Risk:         model.RiskHigh,
			Selected:     selected,
			RequiresRoot: adapter.RequiresRoot,
			LastModified: time.Now().UTC(),
			Result:       result,
		})
	}
	for i := range items {
		if strings.HasPrefix(items[i].RuleID, "uninstall.pkg.") {
			continue
		}
		if _, err := osStat(items[i].Path); errors.Is(err, os.ErrNotExist) {
			items[i].Selected = false
			items[i].Result = "skipped"
		}
	}

	selected := 0
	for _, item := range items {
		if item.Selected {
			selected++
		}
	}

	errCount := 0
	if opts.Apply {
		if err := common.RequireConfirmationOrDryRun(app.Options, "uninstall"); err != nil {
			return model.CommandResult{}, err
		}
		if !app.Options.DryRun {
			for i := range items {
				if !items[i].Selected {
					if err := common.LogApplySkip(ctx, app.Logger, "plan-uninstall", "uninstall", items[i]); err != nil {
						errCount++
					}
					continue
				}
				if strings.HasPrefix(items[i].RuleID, "uninstall.pkg.") {
					entry := model.OperationLogEntry{
						Timestamp: time.Now().UTC(),
						PlanID:    "plan-uninstall",
						Command:   "uninstall",
						Action:    "exec",
						Path:      items[i].Path,
						RuleID:    items[i].RuleID,
						Category:  items[i].Category,
						Risk:      string(items[i].Risk),
						DryRun:    false,
					}

					target, err := parseUninstallTarget(items[i].Path)
					if err != nil {
						items[i].Result = "error"
						errCount++
						entry.Result = items[i].Result
						entry.Error = err.Error()
						if err := app.Logger.Log(ctx, entry); err != nil {
							errCount++
						}
						continue
					}
					adapter, ok := adapterForBackend(target.Backend, adapters)
					if !ok {
						items[i].Result = "error"
						errCount++
						entry.Result = items[i].Result
						entry.Error = "unsupported uninstall backend"
						if err := app.Logger.Log(ctx, entry); err != nil {
							errCount++
						}
						continue
					}
					resolved, ok := availableExecutables[target.Backend]
					if !ok {
						items[i].Result = "skipped"
						entry.Result = items[i].Result
						entry.Error = "backend unavailable"
						if err := app.Logger.Log(ctx, entry); err != nil {
							errCount++
						}
						continue
					}
					if adapter.RequiresRoot && getEUID() != 0 {
						items[i].Result = "skipped"
						entry.Result = items[i].Result
						entry.Error = "requires root"
						if err := app.Logger.Log(ctx, entry); err != nil {
							errCount++
						}
						continue
					}
					cmd := adapter.BuildCommand(target.Name)
					if len(cmd) == 0 {
						items[i].Result = "error"
						errCount++
						entry.Result = items[i].Result
						entry.Error = "invalid uninstall command"
						if err := app.Logger.Log(ctx, entry); err != nil {
							errCount++
						}
						continue
					}
					cmd[0] = resolved
					if err := runCmd(ctx, cmd[0], cmd[1:]...); err != nil {
						items[i].Result = "error"
						errCount++
						entry.Error = err.Error()
					} else {
						items[i].Result = "uninstalled"
					}
					entry.Result = items[i].Result
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}

				entry := model.OperationLogEntry{
					Timestamp: time.Now().UTC(),
					PlanID:    "plan-uninstall",
					Command:   "uninstall",
					Action:    "delete",
					Path:      items[i].Path,
					RuleID:    items[i].RuleID,
					Category:  items[i].Category,
					Risk:      string(items[i].Risk),
					DryRun:    false,
				}
				if !isAllowedUninstallDeletionPath(items[i].Path, home) {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					entry.Error = "path outside uninstall deletion scope"
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}
				if err := pathValidateSystem(items[i].Path, app.Whitelist); err != nil {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}
				if _, err := osStat(items[i].Path); errors.Is(err, os.ErrNotExist) {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}
				if err := safeDelete(items[i].Path, uninstallAllowedRoots(items[i].Path, home), app.Whitelist, false); err != nil {
					items[i].Result = "error"
					errCount++
				} else {
					items[i].Result = "deleted"
				}
				entry.Result = items[i].Result
				if err := app.Logger.Log(ctx, entry); err != nil {
					errCount++
				}
			}
		}
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "uninstall",
		Timestamp:     time.Now().UTC(),
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:    len(items),
			ItemsSelected: selected,
			Errors:        errCount,
		},
		Items: items,
	}, nil
}

func uninstallBinaryTargets(home, executable string, executableErr error) []string {
	targets := []string{
		filepath.Join(home, ".local", "bin", "talpa"),
		"/usr/local/bin/talpa",
	}

	if executableErr == nil {
		trusted, normalized := trustedExecutablePath(executable, targets)
		if trusted {
			targets = append(targets, normalized)
		}
	}

	seen := make(map[string]struct{}, len(targets))
	uniq := make([]string, 0, len(targets))
	for _, p := range targets {
		n := filepath.Clean(p)
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		uniq = append(uniq, n)
	}

	return uniq
}

func trustedExecutablePath(executable string, canonical []string) (bool, string) {
	normalized := filepath.Clean(executable)
	if normalized == "" {
		return false, ""
	}
	if strings.HasPrefix(normalized, os.TempDir()+string(filepath.Separator)) {
		return false, ""
	}
	for _, p := range canonical {
		if normalized == filepath.Clean(p) {
			return true, normalized
		}
	}
	return false, ""
}

func uninstallAdapters() []uninstallAdapter {
	return []uninstallAdapter{
		{Backend: "apt", Executable: "apt-get", RequiresRoot: true, BuildCommand: func(name string) []string { return []string{"apt-get", "remove", "-y", "--", name} }},
		{Backend: "dnf", Executable: "dnf", RequiresRoot: true, BuildCommand: func(name string) []string { return []string{"dnf", "remove", "-y", "--", name} }},
		{Backend: "pacman", Executable: "pacman", RequiresRoot: true, BuildCommand: func(name string) []string { return []string{"pacman", "-Rns", "--noconfirm", "--", name} }},
		{Backend: "zypper", Executable: "zypper", RequiresRoot: true, BuildCommand: func(name string) []string { return []string{"zypper", "remove", "-y", "--", name} }},
		{Backend: "snap", Executable: "snap", RequiresRoot: true, BuildCommand: func(name string) []string { return []string{"snap", "remove", "--purge", "--", name} }},
		{Backend: "flatpak", Executable: "flatpak", RequiresRoot: false, BuildCommand: func(name string) []string { return []string{"flatpak", "uninstall", "--delete-data", "-y", "--", name} }},
	}
}

func adapterForBackend(backend string, adapters []uninstallAdapter) (uninstallAdapter, bool) {
	for _, a := range adapters {
		if a.Backend == backend {
			return a, true
		}
	}
	return uninstallAdapter{}, false
}

func parseUninstallTargets(raw []string) ([]uninstallTarget, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]uninstallTarget, 0, len(raw))
	for _, v := range raw {
		t, err := parseUninstallTarget(v)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func parseUninstallTarget(v string) (uninstallTarget, error) {
	parts := strings.SplitN(strings.TrimSpace(v), ":", 2)
	if len(parts) != 2 {
		return uninstallTarget{}, fmt.Errorf("invalid target %q: expected backend:name", v)
	}
	backend := strings.ToLower(strings.TrimSpace(parts[0]))
	name := strings.TrimSpace(parts[1])
	if backend == "" || name == "" {
		return uninstallTarget{}, fmt.Errorf("invalid target %q: backend and name are required", v)
	}
	if strings.HasPrefix(name, "-") {
		return uninstallTarget{}, fmt.Errorf("invalid target %q: package name must not start with '-'", v)
	}
	allowed := map[string]struct{}{
		"apt":     {},
		"dnf":     {},
		"pacman":  {},
		"zypper":  {},
		"snap":    {},
		"flatpak": {},
	}
	if _, ok := allowed[backend]; !ok {
		return uninstallTarget{}, fmt.Errorf("invalid backend %q: supported backends are apt,dnf,pacman,zypper,snap,flatpak", backend)
	}
	if !isValidTargetNameForBackend(backend, name) {
		return uninstallTarget{}, fmt.Errorf("invalid target %q: unsupported package name for backend %q", v, backend)
	}
	return uninstallTarget{Backend: backend, Name: name}, nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	abs, err := absPath(name)
	if err != nil {
		return err
	}
	resolved, err := evalSymlinks(abs)
	if err != nil {
		return fmt.Errorf("cannot resolve executable symlink: %w", err)
	}
	if !isTrustedExecutablePath(resolved) {
		return fmt.Errorf("untrusted executable path: %s", resolved)
	}
	if err := validateTrustedExecutableFile(resolved); err != nil {
		return err
	}
	cmdCtx, cancel := withDefaultTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, resolved, args...)
	return cmd.Run()
}

func withDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func resolveTrustedExecutable(name string) (string, error) {
	resolved, err := lookPath(name)
	if err != nil {
		return "", err
	}
	abs, err := absPath(resolved)
	if err != nil {
		return "", err
	}
	if !isTrustedExecutablePath(abs) {
		return "", fmt.Errorf("untrusted executable path: %s", abs)
	}
	resolvedPath, err := evalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("cannot resolve executable symlink: %w", err)
	}
	if !isTrustedExecutablePath(resolvedPath) {
		return "", fmt.Errorf("untrusted executable path: %s", resolvedPath)
	}
	abs = resolvedPath
	if err := validateTrustedExecutableFile(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func validateTrustedExecutableFile(path string) error {
	fi, err := osStat(path)
	if err != nil {
		return err
	}
	if fi == nil {
		return fmt.Errorf("untrusted executable metadata: %s", path)
	}
	if fi.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("untrusted executable permissions: %s", path)
	}
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot verify executable owner: %s", path)
	}
	if stat.Uid != 0 {
		return fmt.Errorf("untrusted executable owner: %s", path)
	}
	return nil
}

func isTrustedExecutablePath(path string) bool {
	n := filepath.Clean(path)
	trustedPrefixes := []string{
		"/usr/bin/",
		"/usr/sbin/",
		"/bin/",
		"/sbin/",
		"/usr/local/bin/",
		"/usr/local/sbin/",
		"/snap/bin/",
	}
	for _, p := range trustedPrefixes {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

func isValidTargetNameForBackend(backend, name string) bool {
	if !isCommonTargetName(name) {
		return false
	}

	switch backend {
	case "snap":
		return isValidSnapTargetName(name)
	case "flatpak":
		return isValidFlatpakTargetName(name)
	default:
		return isValidPkgTargetName(name)
	}
}

func isCommonTargetName(name string) bool {
	for _, r := range name {
		if r <= 31 || r == 127 {
			return false
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return false
		}
	}
	return true
}

func isValidPkgTargetName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				return false
			}
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '.', '_', '+', ':', '@', '-', '~', '=':
			continue
		default:
			return false
		}
	}
	return true
}

func isValidSnapTargetName(name string) bool {
	if len(name) < 2 || len(name) > 40 {
		return false
	}
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		if r != '-' {
			return false
		}
		if i == 0 || i == len(name)-1 {
			return false
		}
	}
	if strings.Contains(name, "--") {
		return false
	}
	first := name[0]
	return first >= 'a' && first <= 'z'
}

func isValidFlatpakTargetName(name string) bool {
	segments := strings.Split(name, "/")
	if len(segments) < 1 || len(segments) > 3 {
		return false
	}
	for _, seg := range segments {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		for _, r := range seg {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				continue
			}
			switch r {
			case '.', '_', '-':
				continue
			default:
				return false
			}
		}
	}
	return true
}

func discoverUninstallArtifacts(home string, startIndex int) []model.CandidateItem {
	seen := map[string]struct{}{
		filepath.Clean(filepath.Join(home, ".local", "bin", "talpa")): {},
		filepath.Clean("/usr/local/bin/talpa"):                        {},
		filepath.Clean(filepath.Join(home, ".config", "talpa")):       {},
		filepath.Clean(filepath.Join(home, ".cache", "talpa")):        {},
	}

	type discovered struct {
		ruleID       string
		path         string
		category     string
		risk         model.RiskLevel
		requiresRoot bool
	}
	out := make([]discovered, 0, 16)

	add := func(ruleID, path, category string, risk model.RiskLevel, requiresRoot bool) {
		n := filepath.Clean(path)
		if n == "." || n == "" {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		out = append(out, discovered{
			ruleID:       ruleID,
			path:         n,
			category:     category,
			risk:         risk,
			requiresRoot: requiresRoot,
		})
	}

	for _, d := range []struct {
		dir          string
		ruleID       string
		requiresRoot bool
	}{
		{dir: filepath.Join(home, ".local", "share", "applications"), ruleID: "uninstall.desktop.user", requiresRoot: false},
	} {
		entries, err := osReadDir(d.dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := strings.ToLower(e.Name())
			if !strings.HasSuffix(name, ".desktop") || !strings.Contains(name, "talpa") {
				continue
			}
			add(d.ruleID, filepath.Join(d.dir, e.Name()), "desktop_entry", model.RiskMedium, d.requiresRoot)
		}
	}

	for _, root := range []string{
		filepath.Join(home, ".local", "share"),
		filepath.Join(home, ".local", "state"),
		filepath.Join(home, ".config"),
		filepath.Join(home, ".cache"),
	} {
		entries, err := osReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !isTalpaLeftoverName(name) {
				continue
			}
			add("uninstall.leftover", filepath.Join(root, name), "leftover", model.RiskMedium, false)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ruleID == out[j].ruleID {
			return out[i].path < out[j].path
		}
		return out[i].ruleID < out[j].ruleID
	})

	items := make([]model.CandidateItem, 0, len(out))
	for i, d := range out {
		items = append(items, model.CandidateItem{
			ID:           fmt.Sprintf("uninstall-%d", startIndex+i),
			RuleID:       d.ruleID,
			Path:         d.path,
			Category:     d.category,
			Risk:         d.risk,
			Selected:     true,
			RequiresRoot: d.requiresRoot,
			LastModified: time.Now().UTC(),
			Result:       "planned",
		})
	}

	return items
}

func isTalpaLeftoverName(name string) bool {
	v := strings.ToLower(strings.TrimSpace(name))
	if v == "" {
		return false
	}
	if v == "talpa" {
		return true
	}
	for _, p := range []string{"talpa-", "talpa_", "talpa."} {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	return false
}

func isAllowedUninstallDeletionPath(path, home string) bool {
	n := filepath.Clean(path)
	h := filepath.Clean(home)

	canonical := map[string]struct{}{
		filepath.Clean("/usr/local/bin/talpa"):                     {},
		filepath.Clean(filepath.Join(h, ".local", "bin", "talpa")): {},
		filepath.Clean(filepath.Join(h, ".config", "talpa")):       {},
		filepath.Clean(filepath.Join(h, ".cache", "talpa")):        {},
	}
	if _, ok := canonical[n]; ok {
		return true
	}

	userDesktopDir := filepath.Clean(filepath.Join(h, ".local", "share", "applications"))
	if strings.HasPrefix(n, userDesktopDir+string(filepath.Separator)) {
		base := strings.ToLower(filepath.Base(n))
		if strings.HasSuffix(base, ".desktop") && strings.Contains(base, "talpa") {
			return true
		}
	}

	for _, root := range []string{
		filepath.Clean(filepath.Join(h, ".local", "share")),
		filepath.Clean(filepath.Join(h, ".local", "state")),
		filepath.Clean(filepath.Join(h, ".config")),
		filepath.Clean(filepath.Join(h, ".cache")),
	} {
		if strings.HasPrefix(n, root+string(filepath.Separator)) && isTalpaLeftoverName(filepath.Base(n)) {
			return true
		}
	}

	return false
}

func uninstallAllowedRoots(path, home string) []string {
	n := filepath.Clean(path)
	h := filepath.Clean(home)
	if n == filepath.Clean("/usr/local/bin/talpa") {
		return []string{"/usr/local/bin"}
	}
	for _, root := range []string{
		filepath.Clean(filepath.Join(h, ".local", "bin")),
		filepath.Clean(filepath.Join(h, ".local", "share", "applications")),
		filepath.Clean(filepath.Join(h, ".local", "share")),
		filepath.Clean(filepath.Join(h, ".local", "state")),
		filepath.Clean(filepath.Join(h, ".config")),
		filepath.Clean(filepath.Join(h, ".cache")),
	} {
		if n == root || strings.HasPrefix(n, root+string(filepath.Separator)) {
			return []string{root}
		}
	}
	if dir := filepath.Dir(n); dir != "." && dir != "/" {
		return []string{dir}
	}
	return nil
}

func newPlanItem(id, ruleID, path, category string, risk model.RiskLevel) model.CandidateItem {
	return model.CandidateItem{
		ID:           id,
		RuleID:       ruleID,
		Path:         path,
		Category:     category,
		Risk:         risk,
		Selected:     true,
		RequiresRoot: false,
		LastModified: time.Now().UTC(),
		Result:       "planned",
	}
}
