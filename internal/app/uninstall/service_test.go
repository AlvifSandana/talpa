package uninstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/infra/logging"
)

type captureUninstallLogger struct {
	entries []model.OperationLogEntry
}

func (c *captureUninstallLogger) Log(_ context.Context, entry model.OperationLogEntry) error {
	c.entries = append(c.entries, entry)
	return nil
}

func TestRunApplyRequiresConfirmation(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: false}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestRunPlan(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Command != "uninstall" {
		t.Fatalf("unexpected command %s", res.Command)
	}
	if len(res.Items) == 0 {
		t.Fatalf("expected plan items")
	}
}

func TestRunPlanMarksMissingLocalArtifactsSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	savedStat := osStat
	defer func() { osStat = savedStat }()
	osStat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range res.Items {
		if item.RuleID == "uninstall.pkg.apt" || item.RuleID == "uninstall.pkg.dnf" || item.RuleID == "uninstall.pkg.pacman" || item.RuleID == "uninstall.pkg.zypper" || item.RuleID == "uninstall.pkg.snap" || item.RuleID == "uninstall.pkg.flatpak" {
			continue
		}
		if item.Selected {
			t.Fatalf("expected missing local uninstall item to be unselected: %s", item.RuleID)
		}
		if item.Result != "skipped" {
			t.Fatalf("expected missing local uninstall item skipped, got %s", item.Result)
		}
	}
}

func TestRunApplyExecutesWithConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary := filepath.Join(home, ".local", "bin", "talpa")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedExe := osExecutable
	defer func() { osExecutable = savedExe }()
	osExecutable = func() (string, error) { return binary, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 4 {
		t.Fatalf("expected 4 uninstall items, got %d", len(res.Items))
	}
	for _, item := range res.Items {
		switch item.RuleID {
		case "uninstall.binary", "uninstall.binary.system":
			if item.Result != "deleted" && item.Result != "skipped" {
				t.Fatalf("expected binary item deleted or skipped, got %s", item.Result)
			}
		case "uninstall.config", "uninstall.cache":
			if item.Result != "skipped" {
				t.Fatalf("expected non-whitelisted item skipped, got %s", item.Result)
			}
		}
	}
}

func TestRunApplyDeleteFailureSetsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary := filepath.Join(home, ".local", "bin", "talpa")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedExe := osExecutable
	savedDelete := safeDelete
	defer func() {
		osExecutable = savedExe
		safeDelete = savedDelete
	}()
	osExecutable = func() (string, error) { return binary, nil }
	safeDelete = func(path string, allowedRoots []string, whitelist []string, dryRun bool) error {
		if path == binary {
			return errors.New("delete failed")
		}
		return nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Result != "error" {
		t.Fatalf("expected error, got %s", res.Items[0].Result)
	}
}

func TestRunApplySkipsWhenPathMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary := filepath.Join(home, ".local", "bin", "talpa")

	savedExe := osExecutable
	defer func() { osExecutable = savedExe }()
	osExecutable = func() (string, error) { return binary, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Result != "skipped" {
		t.Fatalf("expected missing binary to be skipped, got %s", res.Items[0].Result)
	}
}

func TestRunApplySkipsOnValidationFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary := filepath.Join(home, ".local", "bin", "talpa")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedExe := osExecutable
	savedValidate := pathValidateSystem
	defer func() {
		osExecutable = savedExe
		pathValidateSystem = savedValidate
	}()
	osExecutable = func() (string, error) { return binary, nil }
	pathValidateSystem = func(path string, whitelist []string) error {
		if path == binary {
			return errors.New("blocked")
		}
		return nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Result != "skipped" {
		t.Fatalf("expected validation failure to skip item, got %s", res.Items[0].Result)
	}
}

func TestRunApplyExecutesAptTarget(t *testing.T) {
	savedRun := runCmd
	savedUID := getEUID
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedResolveExec := resolveExec
	defer func() {
		runCmd = savedRun
		getEUID = savedUID
		lookPath = savedLookPath
		absPath = savedAbsPath
		resolveExec = savedResolveExec
	}()

	called := false
	runCmd = func(ctx context.Context, name string, args ...string) error {
		called = true
		if name != "/usr/bin/apt-get" {
			t.Fatalf("expected /usr/bin/apt-get command, got %s", name)
		}
		wantArgs := []string{"remove", "-y", "--", "vim"}
		if !reflect.DeepEqual(args, wantArgs) {
			t.Fatalf("unexpected apt args: got %#v want %#v", args, wantArgs)
		}
		return nil
	}
	getEUID = func() int { return 0 }
	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	resolveExec = func(name string) (string, error) { return "/usr/bin/" + name, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true, Targets: []string{"apt:vim"}})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatalf("expected package uninstall command execution")
	}

	found := false
	for _, item := range res.Items {
		if item.RuleID == "uninstall.pkg.apt" {
			found = true
			if item.Result != "uninstalled" {
				t.Fatalf("expected apt target uninstalled, got %s", item.Result)
			}
		}
	}
	if !found {
		t.Fatalf("expected uninstall.pkg.apt item")
	}
}

func TestRunApplySkipsRootRequiredTargetWhenNotRoot(t *testing.T) {
	savedRun := runCmd
	savedUID := getEUID
	savedResolveExec := resolveExec
	defer func() {
		runCmd = savedRun
		getEUID = savedUID
		resolveExec = savedResolveExec
	}()

	runCalled := false
	runCmd = func(ctx context.Context, name string, args ...string) error {
		runCalled = true
		return nil
	}
	getEUID = func() int { return 1000 }
	resolveExec = func(name string) (string, error) { return "/usr/bin/" + name, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true, Targets: []string{"snap:code"}})
	if err != nil {
		t.Fatal(err)
	}
	if runCalled {
		t.Fatalf("did not expect command execution without root")
	}
	for _, item := range res.Items {
		if item.RuleID == "uninstall.pkg.snap" && item.Result != "skipped" {
			t.Fatalf("expected snap target skipped, got %s", item.Result)
		}
	}
}

func TestRunRejectsInvalidTarget(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, Options{Targets: []string{"invalid-format"}})
	if err == nil {
		t.Fatal("expected invalid target parsing error")
	}
}

func TestRunRejectsOptionLikeTargetName(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, Options{Targets: []string{"apt:--purge"}})
	if err == nil {
		t.Fatal("expected invalid target name error")
	}
}

func TestRunRejectsInjectedTargetPayloads(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	payloads := []string{
		"apt:vim test",
		"apt:vim\ttest",
		"apt:vim;rm",
		"apt:$(id)",
		"unknown:pkg",
	}
	for _, payload := range payloads {
		_, err := NewService().Run(context.Background(), app, Options{Targets: []string{payload}})
		if err == nil {
			t.Fatalf("expected parse failure for payload %q", payload)
		}
	}
}

func TestRunRejectsSlashForPackageManagerTargets(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, Options{Targets: []string{"apt:vim/stable"}})
	if err == nil {
		t.Fatal("expected parse failure for apt target with slash")
	}
}

func TestRunRejectsLeadingPunctuationForPackageManagerTargets(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	payloads := []string{
		"apt:.vim",
		"dnf:+vim",
		"pacman:-vim",
	}
	for _, payload := range payloads {
		_, err := NewService().Run(context.Background(), app, Options{Targets: []string{payload}})
		if err == nil {
			t.Fatalf("expected parse failure for leading punctuation target %q", payload)
		}
	}
}

func TestRunAcceptsFlatpakRefTargets(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, Options{Targets: []string{"flatpak:org.mozilla.firefox/x86_64/stable"}})
	if err != nil {
		t.Fatalf("expected flatpak ref target to be accepted: %v", err)
	}
}

func TestRunRejectsMalformedFlatpakRefTargets(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	payloads := []string{
		"flatpak:org.mozilla.firefox/x86_64/stable/extra",
		"flatpak:org.mozilla.firefox//stable",
		"flatpak:/x86_64/stable",
		"flatpak:../x86_64/stable",
		"flatpak:org.mozilla.firefox/./stable",
		"flatpak:org.mozilla.firefox/x86_64/stable!",
	}
	for _, payload := range payloads {
		_, err := NewService().Run(context.Background(), app, Options{Targets: []string{payload}})
		if err == nil {
			t.Fatalf("expected malformed flatpak ref parse failure for %q", payload)
		}
	}
}

func TestRunAcceptsValidSnapTargets(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, Options{Targets: []string{"snap:code-insiders"}})
	if err != nil {
		t.Fatalf("expected valid snap target to be accepted: %v", err)
	}
}

func TestRunRejectsInvalidSnapTargets(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	payloads := []string{
		"snap:Code",
		"snap:-code",
		"snap:code--insiders",
		"snap:c",
	}
	for _, payload := range payloads {
		_, err := NewService().Run(context.Background(), app, Options{Targets: []string{payload}})
		if err == nil {
			t.Fatalf("expected invalid snap target parse failure for %q", payload)
		}
	}
}

func TestRunApplyTargetCommandFailureSetsError(t *testing.T) {
	savedRun := runCmd
	savedUID := getEUID
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedResolveExec := resolveExec
	defer func() {
		runCmd = savedRun
		getEUID = savedUID
		lookPath = savedLookPath
		absPath = savedAbsPath
		resolveExec = savedResolveExec
	}()

	runCmd = func(ctx context.Context, name string, args ...string) error { return errors.New("failed") }
	getEUID = func() int { return 0 }
	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	resolveExec = func(name string) (string, error) { return "/usr/bin/" + name, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true, Targets: []string{"zypper:foo"}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range res.Items {
		if item.RuleID == "uninstall.pkg.zypper" {
			found = true
			if item.Result != "error" {
				t.Fatalf("expected zypper target error, got %s", item.Result)
			}
		}
	}
	if !found {
		t.Fatalf("expected uninstall.pkg.zypper item")
	}
}

func TestWithDefaultTimeoutSetsDeadlineWhenMissing(t *testing.T) {
	ctx, cancel := withDefaultTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("expected deadline to be set")
	}
}

func TestWithDefaultTimeoutKeepsExistingDeadline(t *testing.T) {
	base, baseCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer baseCancel()

	ctx, cancel := withDefaultTimeout(base, time.Minute)
	defer cancel()

	baseDeadline, ok := base.Deadline()
	if !ok {
		t.Fatal("expected base deadline")
	}
	ctxDeadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected ctx deadline")
	}
	if !ctxDeadline.Equal(baseDeadline) {
		t.Fatalf("expected existing deadline to be preserved")
	}
}

func TestResolveTrustedExecutableRejectsUntrustedPath(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
	}()

	lookPath = func(file string) (string, error) { return "/tmp/fake-" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }

	_, err := resolveTrustedExecutable("apt-get")
	if err == nil {
		t.Fatal("expected untrusted path error")
	}
}

func TestRunPlanMarksUnavailableUninstallTargetSkipped(t *testing.T) {
	savedResolveExec := resolveExec
	savedStat := osStat
	defer func() { resolveExec = savedResolveExec }()
	defer func() { osStat = savedStat }()
	osStat = func(name string) (os.FileInfo, error) { return nil, nil }

	resolveExec = func(name string) (string, error) {
		if name == "dnf" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + name, nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Targets: []string{"dnf:vim"}})
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, item := range res.Items {
		if item.RuleID == "uninstall.pkg.dnf" {
			found = true
			if item.Selected {
				t.Fatalf("expected unavailable backend target to be unselected")
			}
			if item.Result != "skipped" {
				t.Fatalf("expected unavailable backend target skipped, got %s", item.Result)
			}
		}
	}
	if !found {
		t.Fatalf("expected uninstall.pkg.dnf item")
	}

	if res.Summary.ItemsSelected != len(res.Items)-1 {
		t.Fatalf("expected selected items to exclude unavailable target, got %d selected out of %d", res.Summary.ItemsSelected, len(res.Items))
	}
}

func TestRunApplySkipsUnavailableBackendTarget(t *testing.T) {
	savedRun := runCmd
	savedResolveExec := resolveExec
	defer func() {
		runCmd = savedRun
		resolveExec = savedResolveExec
	}()

	runCalled := false
	runCmd = func(ctx context.Context, name string, args ...string) error {
		runCalled = true
		return nil
	}
	resolveExec = func(name string) (string, error) {
		if name == "zypper" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + name, nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true, Targets: []string{"zypper:foo"}})
	if err != nil {
		t.Fatal(err)
	}
	if runCalled {
		t.Fatalf("did not expect command execution when backend executable is unavailable")
	}

	for _, item := range res.Items {
		if item.RuleID == "uninstall.pkg.zypper" && item.Result != "skipped" {
			t.Fatalf("expected unavailable backend target skipped, got %s", item.Result)
		}
	}
}

func TestRunApplyLogsUnselectedItemsAsSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	savedStat := osStat
	savedResolveExec := resolveExec
	defer func() {
		osStat = savedStat
		resolveExec = savedResolveExec
	}()
	osStat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	resolveExec = func(name string) (string, error) {
		return "", os.ErrNotExist
	}

	logger := &captureUninstallLogger{}
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logger}
	_, err := NewService().Run(context.Background(), app, Options{Apply: true, Targets: []string{"apt:vim"}})
	if err != nil {
		t.Fatal(err)
	}

	if len(logger.entries) != 5 {
		t.Fatalf("expected 5 skip log entries, got %d", len(logger.entries))
	}
	for _, e := range logger.entries {
		if e.Action != "skip" {
			t.Fatalf("expected skip action, got %s", e.Action)
		}
		if e.Result != "skipped" {
			t.Fatalf("expected skipped result, got %s", e.Result)
		}
	}
}

func TestRunPlanMarksLocalStatErrorAsItemError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary := filepath.Join(home, ".local", "bin", "talpa")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedStat := osStat
	defer func() { osStat = savedStat }()
	osStat = func(name string) (os.FileInfo, error) {
		if name == binary {
			return nil, os.ErrPermission
		}
		return nil, os.ErrNotExist
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, item := range res.Items {
		if item.Path == binary {
			found = true
			if item.Selected {
				t.Fatalf("expected stat-error local item to be unselected")
			}
			if item.Result != "error" {
				t.Fatalf("expected stat-error local item result error, got %s", item.Result)
			}
		}
	}
	if !found {
		t.Fatalf("expected local binary item in plan")
	}
}

func TestRunApplyMarksLocalStatErrorAsItemError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary := filepath.Join(home, ".local", "bin", "talpa")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedExe := osExecutable
	savedStat := osStat
	defer func() {
		osExecutable = savedExe
		osStat = savedStat
	}()
	osExecutable = func() (string, error) { return binary, nil }
	statCount := 0
	osStat = func(name string) (os.FileInfo, error) {
		if name == binary {
			statCount++
			if statCount == 1 {
				return fakeFileInfo{name: filepath.Base(binary), dir: false}, nil
			}
			return nil, os.ErrPermission
		}
		return nil, os.ErrNotExist
	}

	logger := &captureUninstallLogger{}
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{binary}, Logger: logger}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, item := range res.Items {
		if item.Path == binary {
			found = true
			if item.Result != "error" {
				t.Fatalf("expected stat-error binary result error, got %s", item.Result)
			}
		}
	}
	if !found {
		t.Fatalf("expected binary item in apply result")
	}
	if res.Summary.Errors == 0 {
		t.Fatalf("expected summary errors > 0 for stat error")
	}

	logged := false
	for _, e := range logger.entries {
		if e.Path == binary {
			logged = true
			if e.Result != "error" {
				t.Fatalf("expected logged error result, got %s", e.Result)
			}
			if e.Error == "" {
				t.Fatalf("expected logged error message for stat failure")
			}
		}
	}
	if !logged {
		t.Fatalf("expected log entry for stat-error local binary")
	}
}

func TestRunApplyCountsPlanDetectedStatError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary := filepath.Join(home, ".local", "bin", "talpa")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedExe := osExecutable
	savedStat := osStat
	defer func() {
		osExecutable = savedExe
		osStat = savedStat
	}()
	osExecutable = func() (string, error) { return binary, nil }
	osStat = func(name string) (os.FileInfo, error) {
		if name == binary {
			return nil, os.ErrPermission
		}
		return nil, os.ErrNotExist
	}

	logger := &captureUninstallLogger{}
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{binary}, Logger: logger}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}

	if res.Summary.Errors == 0 {
		t.Fatalf("expected summary errors > 0 for plan-detected stat error")
	}

	logged := false
	for _, e := range logger.entries {
		if e.Path == binary && e.Result == "error" {
			logged = true
			if e.Error != os.ErrPermission.Error() {
				t.Fatalf("unexpected logged error for plan-detected stat failure: %q", e.Error)
			}
			if e.Category != "app_binary" {
				t.Fatalf("expected normalized category app_binary, got %q", e.Category)
			}
		}
	}
	if !logged {
		t.Fatalf("expected error log entry for plan-detected stat failure")
	}
}

func TestPlanStatErrorCategoryRoundTrip(t *testing.T) {
	err := os.ErrPermission
	encoded := planStatErrorCategory("app_binary", err)
	if got := extractPlanStatError(encoded); got != err.Error() {
		t.Fatalf("unexpected extracted plan stat error: got %q want %q", got, err.Error())
	}
	if got := stripPlanStatErrorCategory(encoded); got != "app_binary" {
		t.Fatalf("unexpected stripped category: got %q want %q", got, "app_binary")
	}
}

func TestDiscoverUninstallArtifactsFindsDesktopAndLeftovers(t *testing.T) {
	home := t.TempDir()
	entries := map[string][]os.DirEntry{
		filepath.Join(home, ".local", "share", "applications"): {
			fakeDirEntry{name: "talpa.desktop"},
			fakeDirEntry{name: "other.desktop"},
		},
		filepath.Join(home, ".local", "share"): {
			fakeDirEntry{name: "talpa-session"},
			fakeDirEntry{name: "notes"},
		},
		filepath.Join(home, ".local", "state"): {
			fakeDirEntry{name: "talpa.log"},
		},
		filepath.Join(home, ".config"): {
			fakeDirEntry{name: "talpa-profile"},
		},
		filepath.Join(home, ".cache"): {
			fakeDirEntry{name: "talpa-temp"},
		},
	}

	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) {
		if v, ok := entries[name]; ok {
			return v, nil
		}
		return nil, os.ErrNotExist
	}

	items := discoverUninstallArtifacts(home, 5)
	if len(items) != 5 {
		t.Fatalf("expected 5 discovered artifacts, got %d", len(items))
	}

	rules := make(map[string]int)
	for _, it := range items {
		rules[it.RuleID]++
	}
	if rules["uninstall.desktop.user"] != 1 {
		t.Fatalf("expected 1 desktop artifact, got %d", rules["uninstall.desktop.user"])
	}
	if rules["uninstall.leftover"] != 4 {
		t.Fatalf("expected 4 leftover artifacts, got %d", rules["uninstall.leftover"])
	}
}

func TestIsTalpaLeftoverName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "exact talpa", in: "talpa", want: true},
		{name: "dash prefix", in: "talpa-cache", want: true},
		{name: "underscore prefix", in: "talpa_state", want: true},
		{name: "dot prefix", in: "talpa.log", want: true},
		{name: "substring only", in: "my-talpa", want: false},
		{name: "unrelated", in: "notes", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTalpaLeftoverName(tc.in); got != tc.want {
				t.Fatalf("unexpected leftover match for %q: got %v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsAllowedUninstallDeletionPath(t *testing.T) {
	home := "/home/tester"
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "allow user binary", path: "/home/tester/.local/bin/talpa", want: true},
		{name: "allow config canonical", path: "/home/tester/.config/talpa", want: true},
		{name: "allow desktop entry", path: "/home/tester/.local/share/applications/talpa.desktop", want: true},
		{name: "allow leftover", path: "/home/tester/.cache/talpa-temp", want: true},
		{name: "deny generic home path", path: "/home/tester/Documents/talpa.txt", want: false},
		{name: "deny unrelated config", path: "/home/tester/.config/not-talpa", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAllowedUninstallDeletionPath(tc.path, home); got != tc.want {
				t.Fatalf("unexpected allow result for %q: got %v want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestUninstallAllowedRoots(t *testing.T) {
	home := "/home/tester"
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "system binary", path: "/usr/local/bin/talpa", want: "/usr/local/bin"},
		{name: "user binary", path: "/home/tester/.local/bin/talpa", want: "/home/tester/.local/bin"},
		{name: "desktop", path: "/home/tester/.local/share/applications/talpa.desktop", want: "/home/tester/.local/share/applications"},
		{name: "leftover", path: "/home/tester/.cache/talpa-temp", want: "/home/tester/.cache"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			roots := uninstallAllowedRoots(tc.path, home)
			if len(roots) != 1 || roots[0] != tc.want {
				t.Fatalf("unexpected roots for %q: got %#v want [%q]", tc.path, roots, tc.want)
			}
		})
	}
}

func TestResolveTrustedExecutableRejectsWorldWritableBinary(t *testing.T) {
	file := filepath.Join(t.TempDir(), "apt-get")
	if err := os.WriteFile(file, []byte("x"), 0o777); err != nil {
		t.Fatal(err)
	}

	savedLookPath := lookPath
	savedAbsPath := absPath
	savedEval := evalSymlinks
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		evalSymlinks = savedEval
	}()

	lookPath = func(name string) (string, error) { return file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }

	_, err := resolveTrustedExecutable("apt-get")
	if err == nil {
		t.Fatal("expected rejection for writable/untrusted executable")
	}
}

func TestResolveTrustedExecutableRejectsWhenOwnerMetadataUnavailable(t *testing.T) {
	file := filepath.Join(t.TempDir(), "apt-get")
	if err := os.WriteFile(file, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedLookPath := lookPath
	savedAbsPath := absPath
	savedEval := evalSymlinks
	savedStat := osStat
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		evalSymlinks = savedEval
		osStat = savedStat
	}()

	lookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) { return fakeOwnerlessFileInfo{}, nil }

	_, err := resolveTrustedExecutable("apt-get")
	if err == nil {
		t.Fatal("expected rejection when owner metadata unavailable")
	}
}

func TestResolveTrustedExecutableRejectsWhenOwnerNotRoot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "apt-get")
	if err := os.WriteFile(file, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	savedLookPath := lookPath
	savedAbsPath := absPath
	savedEval := evalSymlinks
	savedStat := osStat
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		evalSymlinks = savedEval
		osStat = savedStat
	}()

	lookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) { return fakeNonRootOwnerFileInfo{}, nil }

	_, err := resolveTrustedExecutable("apt-get")
	if err == nil {
		t.Fatal("expected rejection for non-root owner")
	}
}

func TestRunCommandRejectsUntrustedExecutionPath(t *testing.T) {
	err := runCommand(context.Background(), "/tmp/evil-binary")
	if err == nil {
		t.Fatal("expected untrusted execution path rejection")
	}
}

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string { return f.name }
func (f fakeDirEntry) IsDir() bool  { return f.dir }
func (f fakeDirEntry) Type() os.FileMode {
	if f.dir {
		return os.ModeDir
	}
	return 0
}
func (f fakeDirEntry) Info() (os.FileInfo, error) { return fakeFileInfo{name: f.name, dir: f.dir}, nil }

type fakeFileInfo struct {
	name string
	dir  bool
}

func (f fakeFileInfo) Name() string { return f.name }
func (f fakeFileInfo) Size() int64  { return 0 }
func (f fakeFileInfo) Mode() os.FileMode {
	if f.dir {
		return os.ModeDir
	}
	return 0
}
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() interface{}   { return nil }

type fakeNonRootOwnerFileInfo struct{}

func (f fakeNonRootOwnerFileInfo) Name() string       { return "non-root" }
func (f fakeNonRootOwnerFileInfo) Size() int64        { return 0 }
func (f fakeNonRootOwnerFileInfo) Mode() os.FileMode  { return 0o755 }
func (f fakeNonRootOwnerFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeNonRootOwnerFileInfo) IsDir() bool        { return false }
func (f fakeNonRootOwnerFileInfo) Sys() interface{}   { return &syscall.Stat_t{Uid: 1000} }

type fakeOwnerlessFileInfo struct{}

func (f fakeOwnerlessFileInfo) Name() string       { return "ownerless" }
func (f fakeOwnerlessFileInfo) Size() int64        { return 0 }
func (f fakeOwnerlessFileInfo) Mode() os.FileMode  { return 0o755 }
func (f fakeOwnerlessFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeOwnerlessFileInfo) IsDir() bool        { return false }
func (f fakeOwnerlessFileInfo) Sys() interface{}   { return nil }
