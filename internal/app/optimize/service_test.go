package optimize

import (
	"context"
	"errors"
	"os"
	"syscall"
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
	if res.Command != "optimize" {
		t.Fatalf("unexpected command %s", res.Command)
	}
	if len(res.Items) == 0 {
		t.Fatalf("expected plan items")
	}
}

func TestRunApplyMarksPendingWithConfirmation(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedEval := evalSymlinks
	savedStat := osStat
	savedRun := runCmd
	savedUID := getEUID
	savedLowBattery := checkLowBattery
	savedReadOnly := checkRootFSReadOnly
	savedBusy := checkPackageManagerBusyFor
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		evalSymlinks = savedEval
		osStat = savedStat
		runCmd = savedRun
		getEUID = savedUID
		checkLowBattery = savedLowBattery
		checkRootFSReadOnly = savedReadOnly
		checkPackageManagerBusyFor = savedBusy
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) {
		return trustedTestFileInfo{}, nil
	}
	runCmd = func(ctx context.Context, name string, args ...string) error { return nil }
	getEUID = func() int { return 0 }
	checkLowBattery = func() bool { return false }
	checkRootFSReadOnly = func() bool { return false }
	checkPackageManagerBusyFor = func(manager string) bool { return false }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range res.Items {
		if it.Result != "optimized" {
			t.Fatalf("expected optimized, got %s", it.Result)
		}
	}
}

func TestRunApplyCommandFailure(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedEval := evalSymlinks
	savedStat := osStat
	savedRun := runCmd
	savedUID := getEUID
	savedLowBattery := checkLowBattery
	savedReadOnly := checkRootFSReadOnly
	savedBusy := checkPackageManagerBusyFor
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		evalSymlinks = savedEval
		osStat = savedStat
		runCmd = savedRun
		getEUID = savedUID
		checkLowBattery = savedLowBattery
		checkRootFSReadOnly = savedReadOnly
		checkPackageManagerBusyFor = savedBusy
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) {
		return trustedTestFileInfo{}, nil
	}
	runCmd = func(ctx context.Context, name string, args ...string) error { return errors.New("failed") }
	getEUID = func() int { return 0 }
	checkLowBattery = func() bool { return false }
	checkRootFSReadOnly = func() bool { return false }
	checkPackageManagerBusyFor = func(manager string) bool { return false }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Errors == 0 {
		t.Fatalf("expected errors when command execution fails")
	}
}

func TestRunApplySkipsWhenNotRoot(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedEval := evalSymlinks
	savedStat := osStat
	savedRun := runCmd
	savedUID := getEUID
	savedLowBattery := checkLowBattery
	savedReadOnly := checkRootFSReadOnly
	savedBusy := checkPackageManagerBusyFor
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		evalSymlinks = savedEval
		osStat = savedStat
		runCmd = savedRun
		getEUID = savedUID
		checkLowBattery = savedLowBattery
		checkRootFSReadOnly = savedReadOnly
		checkPackageManagerBusyFor = savedBusy
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) {
		return trustedTestFileInfo{}, nil
	}
	called := false
	runCmd = func(ctx context.Context, name string, args ...string) error {
		called = true
		return nil
	}
	getEUID = func() int { return 1000 }
	checkLowBattery = func() bool { return false }
	checkRootFSReadOnly = func() bool { return false }
	checkPackageManagerBusyFor = func(manager string) bool { return false }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatalf("expected no command execution when not root")
	}
	for _, it := range res.Items {
		if it.Selected && it.Result != "skipped" {
			t.Fatalf("expected selected item to be skipped, got %s", it.Result)
		}
	}
}

func TestRunPlanMarksUnavailableAdapterSkipped(t *testing.T) {
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

	lookPath = func(file string) (string, error) {
		if file == "dnf" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + file, nil
	}
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) {
		return trustedTestFileInfo{}, nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.ItemsSelected != 7 {
		t.Fatalf("expected 7 selected adapters, got %d", res.Summary.ItemsSelected)
	}
	for _, it := range res.Items {
		if it.RuleID == "optimize.dnf.clean" {
			if it.Selected {
				t.Fatalf("expected dnf item to be unselected")
			}
			if it.Result != "skipped" {
				t.Fatalf("expected dnf item to be skipped, got %s", it.Result)
			}
		}
	}
}

func TestRunApplySkipsWhenPreflightBlocked(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedEval := evalSymlinks
	savedStat := osStat
	savedRun := runCmd
	savedUID := getEUID
	savedLowBattery := checkLowBattery
	savedReadOnly := checkRootFSReadOnly
	savedBusy := checkPackageManagerBusyFor
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		evalSymlinks = savedEval
		osStat = savedStat
		runCmd = savedRun
		getEUID = savedUID
		checkLowBattery = savedLowBattery
		checkRootFSReadOnly = savedReadOnly
		checkPackageManagerBusyFor = savedBusy
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) {
		return trustedTestFileInfo{}, nil
	}
	called := false
	runCmd = func(ctx context.Context, name string, args ...string) error {
		called = true
		return nil
	}
	getEUID = func() int { return 0 }
	checkLowBattery = func() bool { return true }
	checkRootFSReadOnly = func() bool { return false }
	checkPackageManagerBusyFor = func(manager string) bool { return false }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatalf("expected no command execution when preflight is blocked")
	}
	for _, it := range res.Items {
		if it.Selected && it.Result != "skipped" {
			t.Fatalf("expected selected item to be skipped, got %s", it.Result)
		}
	}
}

func TestOptimizePreflightReasonPriority(t *testing.T) {
	savedLowBattery := checkLowBattery
	savedReadOnly := checkRootFSReadOnly
	savedBusy := checkPackageManagerBusyFor
	defer func() {
		checkLowBattery = savedLowBattery
		checkRootFSReadOnly = savedReadOnly
		checkPackageManagerBusyFor = savedBusy
	}()

	tests := []struct {
		name     string
		low      bool
		readOnly bool
		busy     bool
		want     string
	}{
		{name: "low battery first", low: true, readOnly: true, busy: true, want: "preflight blocked: low battery"},
		{name: "read-only when no battery block", low: false, readOnly: true, busy: true, want: "preflight blocked: root filesystem is read-only"},
		{name: "busy when others clear", low: false, readOnly: false, busy: true, want: "preflight blocked: package manager is busy"},
		{name: "no block", low: false, readOnly: false, busy: false, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkLowBattery = func() bool { return tc.low }
			checkRootFSReadOnly = func() bool { return tc.readOnly }
			checkPackageManagerBusyFor = func(manager string) bool {
				if manager != "apt" {
					return false
				}
				return tc.busy
			}

			got := optimizeGlobalPreflightReason()
			if got == "" {
				got = optimizeAdapterPreflightReason("apt")
			}
			if got != tc.want {
				t.Fatalf("unexpected preflight reason: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestIsPackageManagerProcessNameFor(t *testing.T) {
	tests := []struct {
		name string
		mgr  string
		in   string
		want bool
	}{
		{name: "apt-get", mgr: "apt", in: "apt-get", want: true},
		{name: "dpkg", mgr: "apt", in: "dpkg", want: true},
		{name: "packagekitd idle daemon", mgr: "apt", in: "packagekitd", want: false},
		{name: "apt helper not allowed", mgr: "apt", in: "apt.systemd.daily", want: false},
		{name: "dnf helper not allowed", mgr: "dnf", in: "dnf-automatic", want: false},
		{name: "dnf binary", mgr: "dnf", in: "dnf", want: true},
		{name: "zypper", mgr: "zypper", in: "zypper", want: true},
		{name: "unrelated", mgr: "pacman", in: "bash", want: false},
		{name: "empty", mgr: "apt", in: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isPackageManagerProcessNameFor(tc.mgr, tc.in)
			if got != tc.want {
				t.Fatalf("unexpected match result for manager=%q name=%q: got %v want %v", tc.mgr, tc.in, got, tc.want)
			}
		})
	}
}

func TestCmdlineFirstArg(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "normal cmdline", in: "/usr/bin/apt-get\x00update\x00-y\x00", want: "/usr/bin/apt-get"},
		{name: "empty", in: "", want: ""},
		{name: "single token", in: "/usr/bin/dnf", want: "/usr/bin/dnf"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cmdlineFirstArg(tc.in)
			if got != tc.want {
				t.Fatalf("unexpected first arg: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestResolveTrustedExecutableRejectsUntrustedPath(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedStat := osStat
	savedEval := evalSymlinks
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		osStat = savedStat
		evalSymlinks = savedEval
	}()

	lookPath = func(file string) (string, error) { return "/tmp/fake-" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) { return trustedTestFileInfo{}, nil }

	_, err := resolveTrustedExecutable("apt-get")
	if err == nil {
		t.Fatal("expected untrusted path error")
	}
}

func TestResolveTrustedExecutableRejectsWhenOwnerMetadataUnavailable(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedStat := osStat
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		osStat = savedStat
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) { return noOwnerInfoTestFileInfo{}, nil }

	_, err := resolveTrustedExecutable("apt-get")
	if err == nil {
		t.Fatal("expected rejection when owner metadata unavailable")
	}
}

func TestResolveTrustedExecutableRejectsWhenOwnerNotRoot(t *testing.T) {
	savedLookPath := lookPath
	savedAbsPath := absPath
	savedStat := osStat
	savedEval := evalSymlinks
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		osStat = savedStat
		evalSymlinks = savedEval
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	osStat = func(path string) (os.FileInfo, error) { return nonRootOwnerTestFileInfo{}, nil }

	_, err := resolveTrustedExecutable("apt-get")
	if err == nil {
		t.Fatal("expected rejection for non-root owner")
	}
}

func TestRunCommandRejectsUntrustedPathAtExecutionTime(t *testing.T) {
	err := runCommand(context.Background(), "/tmp/evil-binary")
	if err == nil {
		t.Fatal("expected untrusted execution path rejection")
	}
}

type trustedTestFileInfo struct{}

func (trustedTestFileInfo) Name() string       { return "trusted" }
func (trustedTestFileInfo) Size() int64        { return 0 }
func (trustedTestFileInfo) Mode() os.FileMode  { return 0o755 }
func (trustedTestFileInfo) ModTime() time.Time { return time.Time{} }
func (trustedTestFileInfo) IsDir() bool        { return false }
func (trustedTestFileInfo) Sys() interface{}   { return &syscall.Stat_t{Uid: 0} }

type nonRootOwnerTestFileInfo struct{}

func (nonRootOwnerTestFileInfo) Name() string       { return "non-root" }
func (nonRootOwnerTestFileInfo) Size() int64        { return 0 }
func (nonRootOwnerTestFileInfo) Mode() os.FileMode  { return 0o755 }
func (nonRootOwnerTestFileInfo) ModTime() time.Time { return time.Time{} }
func (nonRootOwnerTestFileInfo) IsDir() bool        { return false }
func (nonRootOwnerTestFileInfo) Sys() interface{}   { return &syscall.Stat_t{Uid: 1000} }

type noOwnerInfoTestFileInfo struct{}

func (noOwnerInfoTestFileInfo) Name() string       { return "no-owner" }
func (noOwnerInfoTestFileInfo) Size() int64        { return 0 }
func (noOwnerInfoTestFileInfo) Mode() os.FileMode  { return 0o755 }
func (noOwnerInfoTestFileInfo) ModTime() time.Time { return time.Time{} }
func (noOwnerInfoTestFileInfo) IsDir() bool        { return false }
func (noOwnerInfoTestFileInfo) Sys() interface{}   { return nil }
