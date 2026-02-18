package update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/infra/logging"
)

type alwaysFailUpdateLogger struct{}

func (alwaysFailUpdateLogger) Log(context.Context, model.OperationLogEntry) error {
	return errors.New("log failed")
}

func TestRunDryRun(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if res.Command != "update" {
		t.Fatalf("expected update command, got %s", res.Command)
	}
}

func TestRunRequiresYesWhenNotDryRun(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: false}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app)
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestRunSkipsWhenSourceEqualsTarget(t *testing.T) {
	savedExe := osExecutable
	savedCopy := doCopyFileSafe
	defer func() {
		osExecutable = savedExe
		doCopyFileSafe = savedCopy
	}()

	tmp := filepath.Join(t.TempDir(), "talpa")
	if err := os.WriteFile(tmp, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	osExecutable = func() (string, error) { return tmp, nil }
	doCopyFileSafe = func(src, dst string) error {
		return errors.New("should not copy")
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{tmp}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Items[0].Result; got != "skipped" {
		t.Fatalf("expected skipped, got %s", got)
	}
}

func TestRunUpdateCopyFailure(t *testing.T) {
	savedExe := osExecutable
	savedCopy := doCopyFileSafe
	savedMkdirAll := osMkdirAll
	defer func() {
		osExecutable = savedExe
		doCopyFileSafe = savedCopy
		osMkdirAll = savedMkdirAll
	}()

	osExecutable = func() (string, error) { return "/usr/local/bin/talpa-dev", nil }
	osMkdirAll = func(path string, perm os.FileMode) error { return nil }
	doCopyFileSafe = func(src, dst string) error { return errors.New("copy failed") }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"}, Whitelist: []string{"/usr/local/bin/talpa"}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Items[0].Result; got != "error" {
		t.Fatalf("expected error result, got %s", got)
	}
}

func TestRunIncrementsErrorsWhenLogFails(t *testing.T) {
	savedExe := osExecutable
	savedCopy := doCopyFileSafe
	savedMkdirAll := osMkdirAll
	defer func() {
		osExecutable = savedExe
		doCopyFileSafe = savedCopy
		osMkdirAll = savedMkdirAll
	}()

	osExecutable = func() (string, error) { return "/usr/local/bin/talpa-dev", nil }
	osMkdirAll = func(path string, perm os.FileMode) error { return nil }
	doCopyFileSafe = func(src, dst string) error { return nil }

	app := &common.AppContext{
		Options:   common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"},
		Whitelist: []string{"/usr/local/bin/talpa"},
		Logger:    alwaysFailUpdateLogger{},
	}

	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Summary.Errors; got != 1 {
		t.Fatalf("expected summary errors to include logger failure, got %d", got)
	}
}

func TestRunIncrementsErrorsForOperationAndLogFailure(t *testing.T) {
	savedExe := osExecutable
	savedCopy := doCopyFileSafe
	savedMkdirAll := osMkdirAll
	defer func() {
		osExecutable = savedExe
		doCopyFileSafe = savedCopy
		osMkdirAll = savedMkdirAll
	}()

	osExecutable = func() (string, error) { return "/usr/local/bin/talpa-dev", nil }
	osMkdirAll = func(path string, perm os.FileMode) error { return nil }
	doCopyFileSafe = func(src, dst string) error { return errors.New("copy failed") }

	app := &common.AppContext{
		Options:   common.GlobalOptions{DryRun: false, Yes: true, Confirm: "HIGH-RISK"},
		Whitelist: []string{"/usr/local/bin/talpa"},
		Logger:    alwaysFailUpdateLogger{},
	}

	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Items[0].Result; got != "error" {
		t.Fatalf("expected error result, got %s", got)
	}
	if got := res.Summary.Errors; got != 2 {
		t.Fatalf("expected summary errors to include operation and logger failures, got %d", got)
	}
}
