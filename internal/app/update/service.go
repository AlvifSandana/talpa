package update

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
)

type Service struct{}

var (
	osExecutable   = os.Executable
	osMkdirAll     = os.MkdirAll
	doCopyFileSafe = copyFileAtomic
	osStat         = os.Stat
)

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext) (model.CommandResult, error) {
	exe, err := osExecutable()
	if err != nil {
		return model.CommandResult{}, err
	}
	target := preferredInstallPath(exe)

	item := model.CandidateItem{
		ID:           "update-1",
		RuleID:       "update.binary",
		Path:         target,
		Category:     "self_update",
		Risk:         model.RiskMedium,
		Selected:     true,
		RequiresRoot: strings.HasPrefix(target, "/usr/local/bin/"),
		Result:       "planned",
	}

	errCount := 0
	if !app.Options.DryRun {
		if err := common.RequireConfirmationOrDryRun(app.Options, "update"); err != nil {
			return model.CommandResult{}, err
		}
		if err := common.ValidateSystemScopePath(target, app.Whitelist); err != nil {
			item.Result = "skipped"
			errCount++
		} else if err := osMkdirAll(filepath.Dir(target), 0o755); err != nil {
			item.Result = "error"
			errCount++
		} else {
			if samePath(exe, target) {
				item.Result = "skipped"
			} else if err := doCopyFileSafe(exe, target); err != nil {
				item.Result = "error"
				errCount++
			} else {
				item.Result = "updated"
			}
		}

		if err := app.Logger.Log(ctx, model.OperationLogEntry{
			Timestamp: time.Now().UTC(),
			PlanID:    "plan-update",
			Command:   "update",
			Action:    "exec",
			Path:      target,
			RuleID:    item.RuleID,
			Category:  item.Category,
			Risk:      string(item.Risk),
			Result:    item.Result,
			DryRun:    false,
		}); err != nil {
			errCount++
		}
	}

	if st, err := osStat(exe); err == nil {
		item.SizeBytes = st.Size()
	}
	item.LastModified = time.Now().UTC()
	if app.Options.DryRun {
		item.Result = "planned"
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "update",
		Timestamp:     time.Now().UTC(),
		DurationMS:    0,
		DryRun:        app.Options.DryRun,
		Summary: model.Summary{
			ItemsTotal:    1,
			ItemsSelected: 1,
			Errors:        errCount,
		},
		Items: []model.CandidateItem{item},
	}, nil
}

func preferredInstallPath(exe string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		userTarget := filepath.Join(home, ".local", "bin", "talpa")
		if strings.HasPrefix(exe, filepath.Join(home, ".local", "bin")+string(os.PathSeparator)) {
			return userTarget
		}
	}
	if strings.HasPrefix(exe, "/usr/local/bin/") {
		return "/usr/local/bin/talpa"
	}
	return exe
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, st.Mode())
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
		_ = os.Remove(tmp)
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func samePath(a, b string) bool {
	aa, err := filepath.Abs(a)
	if err != nil {
		return false
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		return false
	}
	return aa == bb
}
