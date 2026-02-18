package cmd

import (
	"strings"
	"testing"
)

func TestPurgeCmdRejectsInvalidDepth(t *testing.T) {
	err := validatePurgeFlags(0, 7)
	if err == nil || !strings.Contains(err.Error(), "--depth must be >= 1") {
		t.Fatalf("expected depth validation error, got %v", err)
	}
}

func TestPurgeCmdRejectsInvalidRecentDays(t *testing.T) {
	err := validatePurgeFlags(4, 0)
	if err == nil || !strings.Contains(err.Error(), "--recent-days must be >= 1") {
		t.Fatalf("expected recent-days validation error, got %v", err)
	}
}

func TestPurgeCmdAcceptsValidFlags(t *testing.T) {
	if err := validatePurgeFlags(4, 7); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
