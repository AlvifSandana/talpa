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
	savedLoad := loadAvgReader
	savedMem := memoryReader
	savedDisk := diskUsageReader
	savedNet := netReader
	savedCPU := cpuUsageReader
	savedTop := topProcessReader
	defer func() {
		loadAvgReader = savedLoad
		memoryReader = savedMem
		diskUsageReader = savedDisk
		netReader = savedNet
		cpuUsageReader = savedCPU
		topProcessReader = savedTop
	}()

	loadAvgReader = func() [3]float64 { return [3]float64{0.11, 0.22, 0.33} }
	memoryReader = func() uint64 { return 4096 }
	diskUsageReader = func(path string) DiskMetric {
		return DiskMetric{Mount: path, UsedBytes: 2048, TotalBytes: 8192}
	}
	netReader = func() NetMetric { return NetMetric{TXBytes: 111, RXBytes: 222} }
	cpuUsageReader = func() float64 { return 0.42 }
	topProcessReader = func(limit int) []system.ProcessStat {
		return []system.ProcessStat{
			{PID: 101, Command: "/usr/bin/vim", CPUPercent: 1.5, MemBytes: 1024},
			{PID: 202, Command: "/usr/bin/go", CPUPercent: 2.5, MemBytes: 2048},
		}
	}

	app := &common.AppContext{Options: common.GlobalOptions{DryRun: true, StatusTop: 2}, Logger: logging.NewNoopLogger()}
	res, err := NewService().Run(context.Background(), app)
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

func normalizeStatusResult(res *model.CommandResult) {
	norm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	res.Timestamp = norm
	res.DurationMS = 0
}
