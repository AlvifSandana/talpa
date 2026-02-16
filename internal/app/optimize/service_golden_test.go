package optimize

import (
	"context"
	"encoding/json"
	"errors"
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
	savedLookPath := lookPath
	savedAbsPath := absPath
	defer func() {
		lookPath = savedLookPath
		absPath = savedAbsPath
	}()

	lookPath = func(file string) (string, error) {
		switch file {
		case "apt-get":
			return "/usr/bin/apt-get", nil
		case "pacman":
			return "/usr/bin/pacman", nil
		default:
			return "", errors.New("not found")
		}
	}
	absPath = func(path string) (string, error) { return path, nil }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}

	normalizeOptimizeResult(&res)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "optimize_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func normalizeOptimizeResult(res *model.CommandResult) {
	norm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	res.Timestamp = norm
	for i := range res.Items {
		res.Items[i].LastModified = norm
	}
}
