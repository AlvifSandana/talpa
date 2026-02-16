package uninstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	lookPath           = exec.LookPath
	absPath            = filepath.Abs
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
				if err := safeDelete(items[i].Path, nil, app.Whitelist, false); err != nil {
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
	cmdCtx, cancel := withDefaultTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, name, args...)
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
	return abs, nil
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
