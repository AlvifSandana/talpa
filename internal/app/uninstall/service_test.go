package uninstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

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
