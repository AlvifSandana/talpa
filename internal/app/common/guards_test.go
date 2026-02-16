package common

import "testing"

func TestRequireConfirmationOrDryRun(t *testing.T) {
	if err := RequireConfirmationOrDryRun(GlobalOptions{DryRun: true}, "x"); err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}
	if err := RequireConfirmationOrDryRun(GlobalOptions{Yes: true}, "x"); err != nil {
		t.Fatalf("unexpected error with --yes: %v", err)
	}
	if err := RequireConfirmationOrDryRun(GlobalOptions{}, "x"); err == nil {
		t.Fatalf("expected confirmation error")
	}
}

func TestValidateSystemScopePath(t *testing.T) {
	if err := ValidateSystemScopePath("/tmp/talpa-test", nil); err != nil {
		t.Fatalf("expected /tmp path allowed with system scope: %v", err)
	}
	if err := ValidateSystemScopePath("/usr/local/bin/talpa", nil); err == nil {
		t.Fatalf("expected blocked validation for system path without whitelist")
	}
	if err := ValidateSystemScopePath("/usr/local/bin/talpa", []string{"/usr/local/bin/talpa"}); err != nil {
		t.Fatalf("expected whitelisted system path allowed: %v", err)
	}
}
