package optimize

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
	lookPath = exec.LookPath
	absPath  = filepath.Abs
	runCmd   = runCommand
	getEUID  = os.Geteuid
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
