package optimize

import (
	"context"
	"errors"
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
	if res.Command != "optimize" {
		t.Fatalf("unexpected command %s", res.Command)
	}
	if len(res.Items) == 0 {
		t.Fatalf("expected plan items")
	}
}

func TestRunApplyMarksPendingWithConfirmation(t *testing.T) {
	savedLookPath := lookPath
	savedRun := runCmd
	savedUID := getEUID
	defer func() {
		lookPath = savedLookPath
		runCmd = savedRun
		getEUID = savedUID
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	runCmd = func(ctx context.Context, name string, args ...string) error { return nil }
	getEUID = func() int { return 0 }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
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
	savedRun := runCmd
	savedUID := getEUID
	defer func() {
		lookPath = savedLookPath
		runCmd = savedRun
		getEUID = savedUID
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	runCmd = func(ctx context.Context, name string, args ...string) error { return errors.New("failed") }
	getEUID = func() int { return 0 }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
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
	savedRun := runCmd
	savedUID := getEUID
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
		runCmd = savedRun
		getEUID = savedUID
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	absPath = func(path string) (string, error) { return path, nil }
	called := false
	runCmd = func(ctx context.Context, name string, args ...string) error {
		called = true
		return nil
	}
	getEUID = func() int { return 1000 }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logging.NewNoopLogger()}
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
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
	}()

	lookPath = func(file string) (string, error) {
		if file == "dnf" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + file, nil
	}
	absPath = func(path string) (string, error) { return path, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.ItemsSelected != 2 {
		t.Fatalf("expected 2 selected adapters, got %d", res.Summary.ItemsSelected)
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
