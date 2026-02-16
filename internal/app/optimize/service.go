package optimize

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
)

type Service struct{}

type Options struct {
	Apply bool
}

type optimizeAdapter struct {
	ID      string
	RuleID  string
	Name    string
	Command []string
}

var (
	lookPath                   = exec.LookPath
	absPath                    = filepath.Abs
	runCmd                     = runCommand
	getEUID                    = os.Geteuid
	checkLowBattery            = isLowBattery
	checkRootFSReadOnly        = isRootFSReadOnly
	checkPackageManagerBusyFor = isPackageManagerBusyFor
)

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, opts Options) (model.CommandResult, error) {
	adapters := optimizeAdapters()

	items := make([]model.CandidateItem, 0, len(adapters))
	for i := range adapters {
		a := adapters[i]
		item := newPlanItem(a.ID, a.RuleID, a.Command[0], "optimization", model.RiskMedium, true)
		resolved, err := lookPath(a.Command[0])
		if err != nil {
			item.Selected = false
			item.Result = "skipped"
		} else if absolute, err := absPath(resolved); err != nil {
			item.Selected = false
			item.Result = "skipped"
		} else {
			adapters[i].Command[0] = absolute
			item.Path = absolute
		}
		items = append(items, item)
	}

	selected := 0
	for _, it := range items {
		if it.Selected {
			selected++
		}
	}

	errCount := 0

	if opts.Apply {
		if err := common.RequireConfirmationOrDryRun(app.Options, "optimize"); err != nil {
			return model.CommandResult{}, err
		}
		if !app.Options.DryRun {
			notRoot := getEUID() != 0
			globalPreflightReason := optimizeGlobalPreflightReason()
			for i := range items {
				if !items[i].Selected {
					continue
				}

				entry := model.OperationLogEntry{
					Timestamp: time.Now().UTC(),
					PlanID:    "plan-optimize",
					Command:   "optimize",
					Action:    "exec",
					Path:      items[i].Path,
					RuleID:    items[i].RuleID,
					Category:  items[i].Category,
					Risk:      string(items[i].Risk),
					DryRun:    false,
				}

				if notRoot {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					entry.Error = "requires root"
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}

				adapter, ok := adapterForRuleID(items[i].RuleID, adapters)
				if !ok || len(adapter.Command) == 0 {
					items[i].Result = "error"
					errCount++
					entry.Result = items[i].Result
					entry.Error = "missing adapter command"
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}

				reason := globalPreflightReason
				if reason == "" {
					reason = optimizeAdapterPreflightReason(adapter.Name)
				}

				if reason != "" {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					entry.Error = reason
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}

				if err := runCmd(ctx, adapter.Command[0], adapter.Command[1:]...); err != nil {
					items[i].Result = "error"
					errCount++
					entry.Error = err.Error()
				} else {
					items[i].Result = "optimized"
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
		Command:       "optimize",
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

func optimizeAdapters() []optimizeAdapter {
	return []optimizeAdapter{
		{ID: "optimize-apt", RuleID: "optimize.apt.clean", Name: "apt", Command: []string{"apt-get", "clean"}},
		{ID: "optimize-dnf", RuleID: "optimize.dnf.clean", Name: "dnf", Command: []string{"dnf", "clean", "all"}},
		{ID: "optimize-pacman", RuleID: "optimize.pacman.clean", Name: "pacman", Command: []string{"pacman", "-Scc", "--noconfirm"}},
		{ID: "optimize-zypper", RuleID: "optimize.zypper.clean", Name: "zypper", Command: []string{"zypper", "clean", "--all"}},
	}
}

func adapterForRuleID(ruleID string, adapters []optimizeAdapter) (optimizeAdapter, bool) {
	for _, a := range adapters {
		if a.RuleID == ruleID {
			return a, true
		}
	}
	return optimizeAdapter{}, false
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, name, args...)
	return cmd.Run()
}

func optimizeGlobalPreflightReason() string {
	if checkLowBattery() {
		return "preflight blocked: low battery"
	}
	if checkRootFSReadOnly() {
		return "preflight blocked: root filesystem is read-only"
	}
	return ""
}

func optimizeAdapterPreflightReason(manager string) string {
	if checkPackageManagerBusyFor(manager) {
		return "preflight blocked: package manager is busy"
	}
	return ""
}

func isLowBattery() bool {
	base := "/sys/class/power_supply"
	entries, err := os.ReadDir(base)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		typeBytes, err := os.ReadFile(filepath.Join(base, e.Name(), "type"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(strings.ToLower(string(typeBytes))) != "battery" {
			continue
		}
		statusBytes, err := os.ReadFile(filepath.Join(base, e.Name(), "status"))
		if err != nil {
			continue
		}
		status := strings.TrimSpace(strings.ToLower(string(statusBytes)))
		if status == "charging" || status == "full" {
			continue
		}

		capBytes, err := os.ReadFile(filepath.Join(base, e.Name(), "capacity"))
		if err != nil {
			continue
		}
		capValue, err := strconv.Atoi(strings.TrimSpace(string(capBytes)))
		if err != nil {
			continue
		}
		if capValue <= 20 {
			return true
		}
	}
	return false
}

func isRootFSReadOnly() bool {
	b, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if fields[1] != "/" {
			continue
		}
		options := strings.Split(fields[3], ",")
		for _, opt := range options {
			if strings.TrimSpace(opt) == "ro" {
				return true
			}
		}
		return false
	}
	return false
}

func isPackageManagerBusyFor(manager string) bool {
	manager = strings.TrimSpace(strings.ToLower(manager))
	if manager == "" {
		return false
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}

		commPath := filepath.Join("/proc", entry.Name(), "comm")
		if b, err := os.ReadFile(commPath); err == nil {
			if isPackageManagerProcessNameFor(manager, strings.TrimSpace(strings.ToLower(string(b)))) {
				return true
			}
		}

		cmdlinePath := filepath.Join("/proc", entry.Name(), "cmdline")
		if b, err := os.ReadFile(cmdlinePath); err == nil {
			firstArg := cmdlineFirstArg(string(b))
			if isPackageManagerProcessNameFor(manager, strings.ToLower(filepath.Base(firstArg))) {
				return true
			}
		}
	}

	return false
}

func isPackageManagerProcessNameFor(manager, name string) bool {
	if name == "" {
		return false
	}

	managerProcesses := map[string]map[string]struct{}{
		"apt": {
			"apt":      {},
			"apt-get":  {},
			"aptitude": {},
			"dpkg":     {},
		},
		"dnf": {
			"dnf": {},
			"yum": {},
			"rpm": {},
		},
		"pacman": {
			"pacman": {},
		},
		"zypper": {
			"zypper": {},
		},
	}

	processes, ok := managerProcesses[manager]
	if !ok {
		return false
	}
	_, matched := processes[name]
	return matched
}

func cmdlineFirstArg(raw string) string {
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "\x00")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func newPlanItem(id, ruleID, path, category string, risk model.RiskLevel, requiresRoot bool) model.CandidateItem {
	return model.CandidateItem{
		ID:           id,
		RuleID:       ruleID,
		Path:         path,
		Category:     category,
		Risk:         risk,
		Selected:     true,
		RequiresRoot: requiresRoot,
		LastModified: time.Now().UTC(),
		Result:       "planned",
	}
}
