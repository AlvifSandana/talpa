package installer

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
	if res.Command != "installer" {
		t.Fatalf("unexpected command %s", res.Command)
	}
	if len(res.Items) == 0 {
		t.Fatalf("expected plan items")
	}
}

func TestRunApplyExecutesWithConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, "Downloads", "talpa-installer.sh")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{target}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 3 {
		t.Fatalf("expected 3 installer items, got %d", len(res.Items))
	}
	for _, item := range res.Items {
		if item.RuleID == "installer.download" {
			if item.Result != "deleted" {
				t.Fatalf("expected download artifact deleted, got %s", item.Result)
			}
			continue
		}
		if item.Result != "skipped" {
			t.Fatalf("expected non-whitelisted artifact skipped, got %s", item.Result)
		}
	}
}

func TestRunApplyDeleteFailureSetsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, "Downloads", "talpa-installer.sh")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	savedDelete := safeDelete
	defer func() { safeDelete = savedDelete }()
	safeDelete = func(path string, allowedRoots []string, whitelist []string, dryRun bool) error {
		if path == target {
			return errors.New("delete failed")
		}
		return nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{target}, Logger: logging.NewNoopLogger()}
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
	target := filepath.Join(home, "Downloads", "talpa-installer.sh")

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{target}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Result != "skipped" {
		t.Fatalf("expected missing target to be skipped, got %s", res.Items[0].Result)
	}
}

func TestRunApplySkipsOnValidationFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, "Downloads", "talpa-installer.sh")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	savedValidate := pathValidateSystem
	defer func() { pathValidateSystem = savedValidate }()
	pathValidateSystem = func(path string, whitelist []string) error {
		if path == target {
			return errors.New("blocked")
		}
		return nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Whitelist: []string{target}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Result != "skipped" {
		t.Fatalf("expected validation failure to skip item, got %s", res.Items[0].Result)
	}
}
