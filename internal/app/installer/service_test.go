package installer

import (
	"context"
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

type captureInstallerLogger struct {
	entries []model.OperationLogEntry
}

func (c *captureInstallerLogger) Log(_ context.Context, entry model.OperationLogEntry) error {
	c.entries = append(c.entries, entry)
	return nil
}

func TestRunApplyRequiresConfirmation(t *testing.T) {
	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: false}, Logger: logging.NewNoopLogger()}
	_, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestRunPlan(t *testing.T) {
	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

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

func TestRunPlanMarksMissingArtifactsSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	savedStat := osStat
	savedReadDir := osReadDir
	defer func() { osStat = savedStat }()
	defer func() { osReadDir = savedReadDir }()
	osStat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.ItemsSelected != 0 {
		t.Fatalf("expected zero selected artifacts when missing, got %d", res.Summary.ItemsSelected)
	}
	for _, item := range res.Items {
		if item.Selected {
			t.Fatalf("expected missing installer artifact to be unselected: %s", item.RuleID)
		}
		if item.Result != "skipped" {
			t.Fatalf("expected missing installer artifact skipped, got %s", item.Result)
		}
	}
}

func TestRunApplyExecutesWithConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

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
	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

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
	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

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
	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

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

func TestRunApplyLogsUnselectedItemsAsSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	savedStat := osStat
	savedReadDir := osReadDir
	defer func() { osStat = savedStat }()
	defer func() { osReadDir = savedReadDir }()
	osStat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	osReadDir = func(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }

	logger := &captureInstallerLogger{}
	app := &common.AppContext{Options: common.GlobalOptions{DryRun: false, Yes: true}, Logger: logger}
	_, err := NewService().Run(context.Background(), app, Options{Apply: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(logger.entries) != 3 {
		t.Fatalf("expected 3 skip log entries, got %d", len(logger.entries))
	}
	for _, e := range logger.entries {
		if e.Action != "skip" {
			t.Fatalf("expected skip action, got %s", e.Action)
		}
		if e.Result != "skipped" {
			t.Fatalf("expected skipped result, got %s", e.Result)
		}
	}
}

func TestDiscoverInstallerArtifactsFindsKnownInstallerPackages(t *testing.T) {
	home := t.TempDir()
	entries := map[string][]os.DirEntry{
		filepath.Join(home, "Downloads"): {
			fakeDirEntry{name: "talpa-installer_1.0.0_amd64.deb"},
			fakeDirEntry{name: "talpa-installer-v1.2.3.tar.gz"},
			fakeDirEntry{name: "talpa-notes.zip"},
		},
		filepath.Join(home, "Desktop"): {
			fakeDirEntry{name: "talpa_installer_latest.AppImage"},
		},
		"/tmp": {
			fakeDirEntry{name: "talpa-installer.run"},
		},
		"/var/tmp": {
			fakeDirEntry{name: "other.tar.gz"},
		},
	}

	savedReadDir := osReadDir
	defer func() { osReadDir = savedReadDir }()
	osReadDir = func(name string) ([]os.DirEntry, error) {
		if v, ok := entries[name]; ok {
			return v, nil
		}
		return nil, os.ErrNotExist
	}

	items := discoverInstallerArtifacts(home)
	if len(items) < 6 {
		t.Fatalf("expected at least seeded artifacts + discovered installers, got %d", len(items))
	}

	var foundDeb, foundTarGz, foundAppImage, foundRun bool
	for _, it := range items {
		switch {
		case strings.HasSuffix(strings.ToLower(it.Path), ".deb"):
			foundDeb = true
		case strings.HasSuffix(strings.ToLower(it.Path), ".tar.gz"):
			foundTarGz = true
		case strings.HasSuffix(strings.ToLower(it.Path), ".appimage"):
			foundAppImage = true
		case strings.HasSuffix(strings.ToLower(it.Path), ".run"):
			foundRun = true
		}
		if strings.Contains(strings.ToLower(filepath.Base(it.Path)), "talpa-notes") {
			t.Fatalf("unexpected non-installer artifact discovered: %s", it.Path)
		}
	}
	if !foundDeb || !foundTarGz || !foundAppImage || !foundRun {
		t.Fatalf("expected deb/tar.gz/appimage/run installer artifacts to be discovered")
	}
}

func TestIsAllowedInstallerDeletionPath(t *testing.T) {
	home := "/home/tester"
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "allow seeded download script", path: "/home/tester/Downloads/talpa-installer.sh", want: true},
		{name: "allow seeded download checksum", path: "/home/tester/Downloads/talpa-installer.sh.sha256", want: true},
		{name: "allow prefixed download package", path: "/home/tester/Downloads/talpa-installer_1.0.0_amd64.deb", want: true},
		{name: "deny prefixed unsupported extension", path: "/home/tester/Downloads/talpa-installer-report.txt", want: false},
		{name: "deny non installer file", path: "/home/tester/Downloads/notes.zip", want: false},
		{name: "allow seeded tmp exact", path: "/tmp/talpa-installer", want: true},
		{name: "allow tmp installer", path: "/tmp/talpa-installer.run", want: true},
		{name: "deny tmp random", path: "/tmp/random.run", want: false},
		{name: "deny traversal outside allowed roots", path: "/home/tester/Downloads/../../etc/talpa-installer.deb", want: false},
		{name: "deny leading whitespace filename", path: "/home/tester/Downloads/ talpa-installer.deb", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAllowedInstallerDeletionPath(tc.path, home); got != tc.want {
				t.Fatalf("unexpected allow result for %q: got %v want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestIsTalpaInstallerFileName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "exact", in: "talpa-installer", want: true},
		{name: "dash suffix", in: "talpa-installer-v1.2.3.tar.gz", want: true},
		{name: "underscore suffix", in: "talpa_installer_latest.AppImage", want: true},
		{name: "numeric suffix", in: "talpainstaller2.run", want: true},
		{name: "dot suffix", in: "talpainstaller.pkg.tar.zst", want: true},
		{name: "deny v-word continuation", in: "talpainstallervirus.zip", want: false},
		{name: "deny word continuation", in: "talpainstallerdocs.zip", want: false},
		{name: "deny unrelated", in: "talpa-notes.zip", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTalpaInstallerFileName(tc.in); got != tc.want {
				t.Fatalf("unexpected installer file match for %q: got %v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestInstallerArtifactRuleID(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantID   string
		wantOkay bool
	}{
		{name: "deb", in: "talpa-installer.deb", wantID: "installer.package.deb", wantOkay: true},
		{name: "appimage case insensitive", in: "talpa-installer.AppImage", wantID: "installer.package.appimage", wantOkay: true},
		{name: "tar.gz", in: "talpa-installer-v1.2.3.tar.gz", wantID: "installer.package.tar.gz", wantOkay: true},
		{name: "tgz", in: "talpa-installer-v1.2.3.tgz", wantID: "installer.package.tar.gz", wantOkay: true},
		{name: "tar.bz2", in: "talpa-installer-v1.2.3.tar.bz2", wantID: "installer.package.tar.bz2", wantOkay: true},
		{name: "tar.xz", in: "talpa-installer-v1.2.3.tar.xz", wantID: "installer.package.tar.xz", wantOkay: true},
		{name: "tar.zst", in: "talpa-installer-v1.2.3.tar.zst", wantID: "installer.package.tar.zst", wantOkay: true},
		{name: "unsupported txt", in: "talpa-installer-report.txt", wantID: "", wantOkay: false},
		{name: "deny leading whitespace", in: " talpa-installer.deb", wantID: "", wantOkay: false},
		{name: "deny trailing whitespace", in: "talpa-installer.deb ", wantID: "", wantOkay: false},
		{name: "empty", in: "", wantID: "", wantOkay: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotOK := installerArtifactRuleID(tc.in)
			if gotOK != tc.wantOkay || gotID != tc.wantID {
				t.Fatalf("unexpected rule id for %q: got (%q,%v) want (%q,%v)", tc.in, gotID, gotOK, tc.wantID, tc.wantOkay)
			}
		})
	}
}

func TestInstallerAllowedRoots(t *testing.T) {
	home := "/home/tester"
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "downloads root", path: "/home/tester/Downloads/talpa-installer.sh", want: "/home/tester/Downloads"},
		{name: "desktop root", path: "/home/tester/Desktop/talpa-installer.zip", want: "/home/tester/Desktop"},
		{name: "tmp root", path: "/tmp/talpa-installer.run", want: "/tmp"},
		{name: "var tmp root", path: "/var/tmp/talpa-installer.run", want: "/var/tmp"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			roots := installerAllowedRoots(tc.path, home)
			if len(roots) != 1 || roots[0] != tc.want {
				t.Fatalf("unexpected roots for %q: got %#v want [%q]", tc.path, roots, tc.want)
			}
		})
	}
}

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string { return f.name }
func (f fakeDirEntry) IsDir() bool  { return f.dir }
func (f fakeDirEntry) Type() os.FileMode {
	if f.dir {
		return os.ModeDir
	}
	return 0
}
func (f fakeDirEntry) Info() (os.FileInfo, error) { return fakeFileInfo{name: f.name, dir: f.dir}, nil }

type fakeFileInfo struct {
	name string
	dir  bool
}

func (f fakeFileInfo) Name() string { return f.name }
func (f fakeFileInfo) Size() int64  { return 0 }
func (f fakeFileInfo) Mode() os.FileMode {
	if f.dir {
		return os.ModeDir
	}
	return 0
}
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() interface{}   { return nil }
