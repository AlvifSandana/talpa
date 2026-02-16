package cmd

import (
	"os"
	"testing"
)

func TestIsCharDevice(t *testing.T) {
	if !isCharDevice(os.ModeCharDevice) {
		t.Fatalf("expected char device mode to be true")
	}
	if isCharDevice(0) {
		t.Fatalf("expected regular mode to be false")
	}
}

func TestIsDumbTerm(t *testing.T) {
	if !isDumbTerm("dumb") {
		t.Fatalf("expected dumb terminal detection")
	}
	if !isDumbTerm("") {
		t.Fatalf("expected empty terminal to be non-interactive")
	}
	if !isDumbTerm(" DUMB ") {
		t.Fatalf("expected normalized dumb terminal detection")
	}
	if isDumbTerm("xterm-256color") {
		t.Fatalf("unexpected dumb terminal detection")
	}
}

func TestShouldUseInteractive(t *testing.T) {
	tests := []struct {
		name     string
		stdin    os.FileMode
		stdout   os.FileMode
		term     string
		expected bool
	}{
		{name: "interactive tty", stdin: os.ModeCharDevice, stdout: os.ModeCharDevice, term: "xterm-256color", expected: true},
		{name: "stdin piped", stdin: 0, stdout: os.ModeCharDevice, term: "xterm-256color", expected: false},
		{name: "stdout redirected", stdin: os.ModeCharDevice, stdout: 0, term: "xterm-256color", expected: false},
		{name: "dumb term", stdin: os.ModeCharDevice, stdout: os.ModeCharDevice, term: "dumb", expected: false},
		{name: "empty term", stdin: os.ModeCharDevice, stdout: os.ModeCharDevice, term: "", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldUseInteractive(tc.stdin, tc.stdout, tc.term); got != tc.expected {
				t.Fatalf("unexpected result: got %v want %v", got, tc.expected)
			}
		})
	}
}
