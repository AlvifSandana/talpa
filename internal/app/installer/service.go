package installer

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
)

type Service struct{}

type Options struct {
	Apply bool
}

var (
	osUserHomeDir      = os.UserHomeDir
	osStat             = os.Stat
	osReadDir          = os.ReadDir
	safeDelete         = safety.SafeDelete
	pathValidateSystem = common.ValidateSystemScopePath
	installerRuleByExt = map[string]string{
		".deb":      "installer.package.deb",
		".rpm":      "installer.package.rpm",
		".appimage": "installer.package.appimage",
		".run":      "installer.package.run",
		".zip":      "installer.package.zip",
		".tar":      "installer.package.tar",
		".gz":       "installer.package.gz",
		".bz2":      "installer.package.bz2",
		".xz":       "installer.package.xz",
		".zst":      "installer.package.zst",
	}
)

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext, opts Options) (model.CommandResult, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return model.CommandResult{}, err
	}

	items := discoverInstallerArtifacts(home)
	for i := range items {
		fi, err := osStat(items[i].Path)
		if errors.Is(err, os.ErrNotExist) {
			items[i].Selected = false
			items[i].Result = "skipped"
			continue
		}
		if err != nil {
			items[i].Selected = false
			items[i].Result = "error"
			continue
		}
		if fi != nil && fi.IsDir() {
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
		if err := common.RequireConfirmationOrDryRun(app.Options, "installer cleanup"); err != nil {
			return model.CommandResult{}, err
		}
		if !app.Options.DryRun {
			for i := range items {
				if !items[i].Selected {
					if err := common.LogApplySkip(ctx, app.Logger, "plan-installer", "installer", items[i]); err != nil {
						errCount++
					}
					continue
				}
				entry := model.OperationLogEntry{
					Timestamp: time.Now().UTC(),
					PlanID:    "plan-installer",
					Command:   "installer",
					Action:    "delete",
					Path:      items[i].Path,
					RuleID:    items[i].RuleID,
					Category:  items[i].Category,
					Risk:      string(items[i].Risk),
					DryRun:    false,
				}
				if !isAllowedInstallerDeletionPath(items[i].Path, home) {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					entry.Error = "path outside installer deletion scope"
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
				fi, err := osStat(items[i].Path)
				if errors.Is(err, os.ErrNotExist) {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}
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
				if fi != nil && fi.IsDir() {
					items[i].Result = "skipped"
					entry.Result = items[i].Result
					entry.Error = "installer artifact must be a file"
					if err := app.Logger.Log(ctx, entry); err != nil {
						errCount++
					}
					continue
				}
				if err := safeDelete(items[i].Path, installerAllowedRoots(items[i].Path, home), app.Whitelist, false); err != nil {
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
		Command:       "installer",
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

func discoverInstallerArtifacts(home string) []model.CandidateItem {
	type artifact struct {
		ruleID   string
		path     string
		category string
		risk     model.RiskLevel
	}

	artifacts := []artifact{
		{ruleID: "installer.download", path: filepath.Join(home, "Downloads", "talpa-installer.sh"), category: "installer_artifact", risk: model.RiskLow},
		{ruleID: "installer.download.sig", path: filepath.Join(home, "Downloads", "talpa-installer.sh.sha256"), category: "installer_artifact", risk: model.RiskLow},
		{ruleID: "installer.tmp", path: filepath.Join("/tmp", "talpa-installer"), category: "installer_artifact", risk: model.RiskLow},
	}

	seen := make(map[string]struct{}, len(artifacts)+16)
	for i := range artifacts {
		n := filepath.Clean(artifacts[i].path)
		artifacts[i].path = n
		seen[n] = struct{}{}
	}

	add := func(ruleID, path, category string, risk model.RiskLevel) {
		n := filepath.Clean(path)
		if n == "." || n == "" {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		artifacts = append(artifacts, artifact{ruleID: ruleID, path: n, category: category, risk: risk})
	}

	for _, dir := range []string{filepath.Join(home, "Downloads"), filepath.Join(home, "Desktop"), "/tmp", "/var/tmp"} {
		entries, err := osReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !isTalpaInstallerFileName(name) {
				continue
			}
			ruleID, ok := installerArtifactRuleID(name)
			if !ok {
				continue
			}
			add(ruleID, filepath.Join(dir, name), "installer_artifact", model.RiskLow)
		}
	}

	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].ruleID == artifacts[j].ruleID {
			return artifacts[i].path < artifacts[j].path
		}
		return artifacts[i].ruleID < artifacts[j].ruleID
	})

	items := make([]model.CandidateItem, 0, len(artifacts))
	for i, a := range artifacts {
		items = append(items, newPlanItem(
			"installer-"+strconv.Itoa(i+1),
			a.ruleID,
			a.path,
			a.category,
			a.risk,
		))
	}
	return items
}

func isAllowedInstallerDeletionPath(path, home string) bool {
	n := filepath.Clean(path)
	h := filepath.Clean(home)

	allowedPrefixes := []string{
		filepath.Join(h, "Downloads") + string(filepath.Separator),
		filepath.Join(h, "Desktop") + string(filepath.Separator),
		"/tmp/",
		"/var/tmp/",
	}

	allowedExact := map[string]struct{}{
		filepath.Clean(filepath.Join(h, "Downloads", "talpa-installer.sh")):        {},
		filepath.Clean(filepath.Join(h, "Downloads", "talpa-installer.sh.sha256")): {},
		filepath.Clean(filepath.Join("/tmp", "talpa-installer")):                   {},
	}
	if _, ok := allowedExact[n]; ok {
		return true
	}
	for _, p := range allowedPrefixes {
		if strings.HasPrefix(n, p) {
			base := filepath.Base(n)
			if !isTalpaInstallerFileName(base) {
				return false
			}
			_, ok := installerArtifactRuleID(base)
			return ok
		}
	}
	return false
}

func installerAllowedRoots(path, home string) []string {
	n := filepath.Clean(path)
	h := filepath.Clean(home)

	for _, root := range []string{
		filepath.Clean(filepath.Join(h, "Downloads")),
		filepath.Clean(filepath.Join(h, "Desktop")),
		filepath.Clean("/tmp"),
		filepath.Clean("/var/tmp"),
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

func isTalpaInstallerFileName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	prefixes := []string{"talpa-installer", "talpa_installer", "talpainstaller"}
	for _, p := range prefixes {
		if !strings.HasPrefix(lower, p) {
			continue
		}
		if len(lower) == len(p) {
			return true
		}
		next := lower[len(p)]
		if (next >= '0' && next <= '9') || next == '-' || next == '_' || next == '.' {
			return true
		}
	}
	return false
}

func installerArtifactRuleID(name string) (string, bool) {
	if strings.TrimSpace(name) != name {
		return "", false
	}
	lower := strings.ToLower(name)
	if lower == "" {
		return "", false
	}

	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return "installer.package.tar.gz", true
	}
	if strings.HasSuffix(lower, ".tar.bz2") {
		return "installer.package.tar.bz2", true
	}
	if strings.HasSuffix(lower, ".tar.xz") {
		return "installer.package.tar.xz", true
	}
	if strings.HasSuffix(lower, ".tar.zst") {
		return "installer.package.tar.zst", true
	}

	ext := filepath.Ext(lower)
	rule, ok := installerRuleByExt[ext]
	return rule, ok
}
