package update

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

	savedExe := osExecutable
	savedStat := osStat
	defer func() {
		osExecutable = savedExe
		osStat = savedStat
	}()

	osExecutable = func() (string, error) { return filepath.Join(home, ".local", "bin", "talpa-dev"), nil }
	osStat = func(name string) (os.FileInfo, error) {
		return fakeFileInfo{name: filepath.Base(name), size: 123, mode: 0o755, mod: time.Now()}, nil
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}

	normalizeUpdateResult(&res, home)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "update_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func normalizeUpdateResult(res *model.CommandResult, home string) {
	norm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	res.Timestamp = norm
	res.DurationMS = 0
	for i := range res.Items {
		res.Items[i].Path = strings.ReplaceAll(res.Items[i].Path, home, "$HOME")
		res.Items[i].LastModified = norm
	}
}

type fakeFileInfo struct {
	name string
	size int64
	mode os.FileMode
	mod  time.Time
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return f.mod }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }
