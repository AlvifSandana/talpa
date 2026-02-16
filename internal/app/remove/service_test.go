package remove

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"talpa/internal/app/common"
	"talpa/internal/infra/logging"
)

func TestRunDryRun(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if res.Command != "remove" {
		t.Fatalf("expected remove command, got %s", res.Command)
	}
}

func TestRunRequiresYesWhenNotDryRun(t *testing.T) {
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: false}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app)
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestRunRemoveSuccess(t *testing.T) {
	savedExe := osExecutable
	savedRemove := osRemove
	defer func() {
		osExecutable = savedExe
		osRemove = savedRemove
	}()

	tmp := filepath.Join(t.TempDir(), "talpa")
	if err := os.WriteFile(tmp, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	osExecutable = func() (string, error) { return tmp, nil }
	osRemove = func(name string) error {
		return os.Remove(name)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Items[0].Result; got != "deleted" {
		t.Fatalf("expected deleted, got %s", got)
	}
}

func TestRunRemoveFailure(t *testing.T) {
	savedExe := osExecutable
	savedRemove := osRemove
	defer func() {
		osExecutable = savedExe
		osRemove = savedRemove
	}()

	osExecutable = func() (string, error) { return "/tmp/talpa", nil }
	osRemove = func(name string) error { return errors.New("permission denied") }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Items[0].Result; got != "error" {
		t.Fatalf("expected error, got %s", got)
	}
}
