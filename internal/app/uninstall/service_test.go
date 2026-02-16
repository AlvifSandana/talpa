package uninstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/infra/logging"
)

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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{binary}, Logger: logging.NewNoopLogger()}
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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
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
	defer func() { resolveExec = savedResolveExec }()

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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
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
