package installer

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

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}

	normalizeInstallerResult(&res, home)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "installer_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func normalizeInstallerResult(res *model.CommandResult, home string) {
	norm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	res.Timestamp = norm
	for i := range res.Items {
		res.Items[i].Path = strings.ReplaceAll(res.Items[i].Path, home, "$HOME")
		res.Items[i].LastModified = norm
	}
}
