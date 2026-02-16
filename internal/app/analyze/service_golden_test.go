package analyze

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

	cacheDir := filepath.Join(home, "workspace", "project", "cache")
	docsDir := filepath.Join(home, "workspace", "project", "docs")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "blob.bin"), []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "readme.md"), []byte("abcd"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, filepath.Join(home, "workspace"), Options{Depth: 6, Limit: 10, SortBy: "path"})
	if err != nil {
		t.Fatal(err)
	}

	normalizeAnalyzeResult(&res, home)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "analyze_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func normalizeAnalyzeResult(res *model.CommandResult, home string) {
	norm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	res.Timestamp = norm
	res.DurationMS = 0
	for i := range res.Items {
		res.Items[i].Path = strings.ReplaceAll(res.Items[i].Path, home, "$HOME")
		res.Items[i].LastModified = norm
	}
}
