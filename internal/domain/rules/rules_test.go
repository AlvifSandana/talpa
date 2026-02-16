package rules

import (
	"testing"
)

func TestCleanRulesNotEmpty(t *testing.T) {
	r := CleanRules("/tmp/home")
	if len(r) == 0 {
		t.Fatal("expected clean rules")
	}
}

func TestPurgeRulesNotEmpty(t *testing.T) {
	r := PurgeArtifactRules()
	if len(r) == 0 {
		t.Fatal("expected purge rules")
	}
}
