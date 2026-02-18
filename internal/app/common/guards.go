package common

import (
	"fmt"
	"strings"

	"talpa/internal/domain/safety"
)

func RequireConfirmationOrDryRun(opts GlobalOptions, action string) error {
	if opts.DryRun || opts.Yes {
		return nil
	}
	return fmt.Errorf("confirmation required for %s: use --yes or --dry-run", action)
}

func RequireHighRiskConfirmationOrDryRun(opts GlobalOptions, action string) error {
	if opts.DryRun {
		return nil
	}
	if !opts.Yes {
		return fmt.Errorf("confirmation required for %s: use --yes --confirm HIGH-RISK or --dry-run", action)
	}
	if !strings.EqualFold(strings.TrimSpace(opts.Confirm), "HIGH-RISK") {
		return fmt.Errorf("double confirmation required for %s: use --yes --confirm HIGH-RISK or --dry-run", action)
	}
	return nil
}

func ValidateSystemScopePath(path string, whitelist []string) error {
	return safety.ValidatePath(path, nil, whitelist)
}
