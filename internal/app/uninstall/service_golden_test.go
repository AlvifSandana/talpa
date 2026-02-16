package uninstall

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/infra/logging"
)

func TestRunDryRunGoldenJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	savedExe := osExecutable
	savedResolveExec := resolveExec
	savedStat := osStat
	defer func() { osExecutable = savedExe }()
	defer func() { resolveExec = savedResolveExec }()
	defer func() { osStat = savedStat }()
	osExecutable = func() (string, error) { return filepath.Join(home, ".local", "bin", "talpa"), nil }
	resolveExec = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	osStat = func(name string) (os.FileInfo, error) { return nil, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}

	normalizeUninstallResult(&res, home)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "uninstall_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func TestRunDryRunTargetsGoldenJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	savedExe := osExecutable
	savedResolveExec := resolveExec
	savedStat := osStat
	defer func() { osExecutable = savedExe }()
	defer func() { resolveExec = savedResolveExec }()
	defer func() { osStat = savedStat }()
	osExecutable = func() (string, error) { return filepath.Join(home, ".local", "bin", "talpa"), nil }
	resolveExec = func(name string) (string, error) {
		if name == "dnf" {
			return "", os.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	osStat = func(name string) (os.FileInfo, error) { return nil, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(
		context.Background(),
		app,
		Options{Targets: []string{"apt:vim", "dnf:vim", "flatpak:org.mozilla.firefox/x86_64/stable"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	normalizeUninstallResult(&res, home)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "uninstall_targets_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func TestRunDryRunMissingArtifactsGoldenJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	savedExe := osExecutable
	savedResolveExec := resolveExec
	savedStat := osStat
	defer func() { osExecutable = savedExe }()
	defer func() { resolveExec = savedResolveExec }()
	defer func() { osStat = savedStat }()
	osExecutable = func() (string, error) { return filepath.Join(home, ".local", "bin", "talpa"), nil }
	resolveExec = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	osStat = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}

	normalizeUninstallResult(&res, home)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "uninstall_missing_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func normalizeUninstallResult(res *model.CommandResult, home string) {
	norm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	res.Timestamp = norm
	for i := range res.Items {
		res.Items[i].Path = strings.ReplaceAll(res.Items[i].Path, home, "$HOME")
		res.Items[i].LastModified = norm
	}
}
