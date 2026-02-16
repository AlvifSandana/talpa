package system

import "testing"

func TestParseTotalCPUJiffies(t *testing.T) {
	line := "cpu  100 20 30 400 50 0 0 0 0 0"
	total, ok := parseTotalCPUJiffies(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if total != 600 {
		t.Fatalf("unexpected total: got %d want %d", total, 600)
	}
}

func TestParseTotalCPUJiffiesRejectsInvalid(t *testing.T) {
	_, ok := parseTotalCPUJiffies("bogus line")
	if ok {
		t.Fatalf("expected parse failure")
	}
}

func TestParseProcCPUJiffies(t *testing.T) {
	line := "1234 (cmd with spaces) R 1 1 1 1 1 1 1 1 1 1 100 50 0 0 0 0 0 0 0 0"
	jiffies, ok := parseProcCPUJiffies(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if jiffies != 150 {
		t.Fatalf("unexpected jiffies: got %d want %d", jiffies, 150)
	}
}

func TestParseProcCPUJiffiesRejectsInvalid(t *testing.T) {
	_, ok := parseProcCPUJiffies("1234 cmd")
	if ok {
		t.Fatalf("expected parse failure")
	}
}

func TestParseProcCPUJiffiesRejectsMissingSeparator(t *testing.T) {
	line := "1234 (cmd)R 1 1 1 1 1 1 1 1 1 1 100 50"
	_, ok := parseProcCPUJiffies(line)
	if ok {
		t.Fatalf("expected parse failure")
	}
}

func TestComputeCPUPercent(t *testing.T) {
	if got := computeCPUPercent(25, 100); got != 25 {
		t.Fatalf("unexpected percent: got %.2f want 25", got)
	}
}

func TestComputeCPUPercentZeroTotal(t *testing.T) {
	if got := computeCPUPercent(10, 0); got != 0 {
		t.Fatalf("unexpected percent: got %.2f want 0", got)
	}
}

func TestComputeCPUPercentClampsHigh(t *testing.T) {
	if got := computeCPUPercent(200, 100); got != 100 {
		t.Fatalf("unexpected percent: got %.2f want 100", got)
	}
}
