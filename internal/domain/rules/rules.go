package rules

import (
	"os"
	"path/filepath"

	"talpa/internal/domain/model"
)

func CleanRules(home string) []model.Rule {
	return []model.Rule{
		{ID: "clean.xdg.cache", Command: "clean", Category: "xdg_cache", Pattern: filepath.Join(home, ".cache"), Risk: model.RiskLow},
		{ID: "clean.trash", Command: "clean", Category: "trash", Pattern: filepath.Join(home, ".local", "share", "Trash"), Risk: model.RiskLow},
		{ID: "clean.thumbnails", Command: "clean", Category: "thumbnails", Pattern: filepath.Join(home, ".cache", "thumbnails"), Risk: model.RiskLow},
		{ID: "clean.browser.chromium", Command: "clean", Category: "browser_cache", Pattern: filepath.Join(home, ".cache", "chromium"), Risk: model.RiskLow},
		{ID: "clean.browser.google_chrome", Command: "clean", Category: "browser_cache", Pattern: filepath.Join(home, ".cache", "google-chrome"), Risk: model.RiskLow},
		{ID: "clean.browser.brave", Command: "clean", Category: "browser_cache", Pattern: filepath.Join(home, ".cache", "BraveSoftware"), Risk: model.RiskLow},
		{ID: "clean.browser.firefox", Command: "clean", Category: "browser_cache", Pattern: filepath.Join(home, ".cache", "mozilla", "firefox"), Risk: model.RiskLow},
		{ID: "clean.electron.slack", Command: "clean", Category: "electron_cache", Pattern: filepath.Join(home, ".config", "Slack", "Cache"), Risk: model.RiskLow},
		{ID: "clean.electron.discord", Command: "clean", Category: "electron_cache", Pattern: filepath.Join(home, ".config", "discord", "Cache"), Risk: model.RiskLow},
		{ID: "clean.electron.vscode", Command: "clean", Category: "electron_cache", Pattern: filepath.Join(home, ".config", "Code", "Cache"), Risk: model.RiskLow},
		{ID: "clean.dev.npm", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".npm"), Risk: model.RiskLow},
		{ID: "clean.dev.yarn", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".cache", "yarn"), Risk: model.RiskLow},
		{ID: "clean.dev.pnpm", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".pnpm-store"), Risk: model.RiskLow},
		{ID: "clean.dev.pip", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".cache", "pip"), Risk: model.RiskLow},
		{ID: "clean.dev.poetry", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".cache", "pypoetry"), Risk: model.RiskLow},
		{ID: "clean.dev.go.build", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".cache", "go-build"), Risk: model.RiskLow},
		{ID: "clean.dev.go.mod", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, "go", "pkg", "mod"), Risk: model.RiskMedium},
		{ID: "clean.dev.cargo", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".cargo", "registry"), Risk: model.RiskMedium},
		{ID: "clean.dev.gradle", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".gradle", "caches"), Risk: model.RiskMedium},
		{ID: "clean.dev.maven", Command: "clean", Category: "dev_cache", Pattern: filepath.Join(home, ".m2", "repository"), Risk: model.RiskMedium},
		{ID: "clean.logs.local_state", Command: "clean", Category: "user_logs", Pattern: filepath.Join(home, ".local", "state"), Risk: model.RiskMedium},
	}
}

func CleanSystemRules() []model.Rule {
	return []model.Rule{
		{ID: "clean.system.tmp", Command: "clean", Category: "system_tmp", Pattern: "/tmp", Risk: model.RiskMedium, RequiresRoot: true},
		{ID: "clean.system.var_tmp", Command: "clean", Category: "system_tmp", Pattern: "/var/tmp", Risk: model.RiskMedium, RequiresRoot: true},
		{ID: "clean.system.apt_cache", Command: "clean", Category: "system_cache", Pattern: "/var/cache/apt", Risk: model.RiskMedium, RequiresRoot: true},
		{ID: "clean.system.dnf_cache", Command: "clean", Category: "system_cache", Pattern: "/var/cache/dnf", Risk: model.RiskMedium, RequiresRoot: true},
		{ID: "clean.system.pacman_cache", Command: "clean", Category: "system_cache", Pattern: "/var/cache/pacman", Risk: model.RiskMedium, RequiresRoot: true},
		{ID: "clean.system.zypper_cache", Command: "clean", Category: "system_cache", Pattern: "/var/cache/zypp", Risk: model.RiskMedium, RequiresRoot: true},
		{ID: "clean.system.journal", Command: "clean", Category: "system_logs", Pattern: "/var/log/journal", Risk: model.RiskHigh, RequiresRoot: true},
	}
}

func PurgeArtifactRules() []model.Rule {
	return []model.Rule{
		{ID: "purge.node_modules", Command: "purge", Category: "project_artifact", Pattern: "node_modules", Risk: model.RiskLow},
		{ID: "purge.node.next", Command: "purge", Category: "project_artifact", Pattern: ".next", Risk: model.RiskLow},
		{ID: "purge.web.dist", Command: "purge", Category: "project_artifact", Pattern: "dist", Risk: model.RiskLow},
		{ID: "purge.web.build", Command: "purge", Category: "project_artifact", Pattern: "build", Risk: model.RiskLow},
		{ID: "purge.rust.target", Command: "purge", Category: "project_artifact", Pattern: "target", Risk: model.RiskLow},
		{ID: "purge.python.venv", Command: "purge", Category: "project_artifact", Pattern: ".venv", Risk: model.RiskLow},
		{ID: "purge.python.venv_alt", Command: "purge", Category: "project_artifact", Pattern: "venv", Risk: model.RiskLow},
		{ID: "purge.python.pycache", Command: "purge", Category: "project_artifact", Pattern: "__pycache__", Risk: model.RiskLow},
		{ID: "purge.python.pytest", Command: "purge", Category: "project_artifact", Pattern: ".pytest_cache", Risk: model.RiskLow},
		{ID: "purge.mobile.dart_tool", Command: "purge", Category: "project_artifact", Pattern: ".dart_tool", Risk: model.RiskLow},
		{ID: "purge.mobile.pods", Command: "purge", Category: "project_artifact", Pattern: "Pods", Risk: model.RiskLow},
		{ID: "purge.mobile.derived_data", Command: "purge", Category: "project_artifact", Pattern: "DerivedData", Risk: model.RiskLow},
	}
}

func ExistingCleanRules(home string, includeSystem bool) []model.Rule {
	all := CleanRules(home)
	if includeSystem {
		all = append(all, CleanSystemRules()...)
	}
	out := make([]model.Rule, 0, len(all))
	for _, r := range all {
		if st, err := os.Stat(r.Pattern); err == nil && st.IsDir() {
			out = append(out, r)
		}
	}
	return out
}
