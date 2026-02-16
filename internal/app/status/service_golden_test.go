package status

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
	"talpa/internal/infra/system"
)

func TestRunDryRunGoldenJSON(t *testing.T) {
	svc := Service{readers: statusReaders{
		loadAvg: func() [3]float64 { return [3]float64{0.11, 0.22, 0.33} },
		memory:  func() uint64 { return 4096 },
		diskUsage: func(path string) DiskMetric {
			return DiskMetric{Mount: path, UsedBytes: 2048, TotalBytes: 8192}
		},
		net:      func() NetMetric { return NetMetric{TXBytes: 111, RXBytes: 222} },
		cpuUsage: func() float64 { return 0.42 },
		topProcess: func(limit int) []system.ProcessStat {
			return []system.ProcessStat{
				{PID: 101, Command: "/usr/bin/vim", CPUPercent: 1.5, MemBytes: 1024},
				{PID: 202, Command: "/usr/bin/go", CPUPercent: 2.5, MemBytes: 2048},
			}
		},
	}}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true, StatusTop: 2}, Logger: logging.NewNoopLogger()}
	res, err := svc.Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}

	normalizeStatusResult(&res)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "status_dry_run.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(b))
	w := strings.TrimSpace(string(want))
	if got != w {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, w)
	}
}

func TestRunWithPartialReadersFallsBackToDefaults(t *testing.T) {
	svc := Service{readers: statusReaders{
		loadAvg: func() [3]float64 { return [3]float64{9.9, 8.8, 7.7} },
	}}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true, StatusTop: 1}, Logger: logging.NewNoopLogger()}
	res, err := svc.Run(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}

	metrics, ok := res.Metrics.(Metrics)
	if !ok {
		t.Fatalf("expected status metrics type, got %T", res.Metrics)
	}
	if metrics.LoadAvg != [3]float64{9.9, 8.8, 7.7} {
		t.Fatalf("expected overridden load avg, got %#v", metrics.LoadAvg)
	}
}

func normalizeStatusResult(res *model.CommandResult) {
	norm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	res.Timestamp = norm
	res.DurationMS = 0
}
