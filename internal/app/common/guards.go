package common

import (
	"fmt"

	"talpa/internal/domain/safety"
)

func RequireConfirmationOrDryRun(opts GlobalOptions, action string) error {
	if opts.DryRun || opts.Yes {
		return nil
	}
	return fmt.Errorf("confirmation required for %s: use --yes or --dry-run", action)
}

func ValidateSystemScopePath(path string, whitelist []string) error {
	return safety.ValidatePath(path, nil, whitelist)
}
