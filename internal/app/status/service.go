package status

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/infra/system"
)

type Service struct{}

type Metrics struct {
	CPUUsage        float64      `json:"cpu_usage"`
	LoadAvg         [3]float64   `json:"load_avg"`
	MemoryUsedBytes uint64       `json:"memory_used_bytes"`
	DiskUsage       []DiskMetric `json:"disk_usage"`
	Net             NetMetric    `json:"net"`
	TopProcesses    []Process    `json:"top_processes"`
}

type Process struct {
	PID        int     `json:"pid"`
	Command    string  `json:"command"`
	CPUPercent float64 `json:"cpu_percent"`
	MemBytes   uint64  `json:"mem_bytes"`
}

type DiskMetric struct {
	Mount      string `json:"mount"`
	UsedBytes  uint64 `json:"used_bytes"`
	TotalBytes uint64 `json:"total_bytes"`
}

type NetMetric struct {
	TXBytes uint64 `json:"tx_bytes"`
	RXBytes uint64 `json:"rx_bytes"`
}

func NewService() Service { return Service{} }

func (Service) Run(ctx context.Context, app *common.AppContext) (model.CommandResult, error) {
	_ = ctx
	start := time.Now()

	load := readLoadAvg()
	mem := readMemoryUsed()
	disk := readDiskUsage("/")
	net := readNetDev()

	top := system.TopProcesses(app.Options.StatusTop)
	procs := make([]Process, 0, len(top))
	for _, p := range top {
		procs = append(procs, Process{PID: p.PID, Command: p.Command, CPUPercent: p.CPUPercent, MemBytes: p.MemBytes})
	}

	metrics := Metrics{
		CPUUsage:        readCPUUsage(),
		LoadAvg:         load,
		MemoryUsedBytes: mem,
		DiskUsage:       []DiskMetric{disk},
		Net:             net,
		TopProcesses:    procs,
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "status",
		Timestamp:     time.Now().UTC(),
		DurationMS:    time.Since(start).Milliseconds(),
		Metrics:       metrics,
	}, nil
}

func readLoadAvg() [3]float64 {
	var out [3]float64
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return out
	}
	parts := strings.Fields(string(b))
	for i := 0; i < 3 && i < len(parts); i++ {
		v, _ := strconv.ParseFloat(parts[i], 64)
		out[i] = v
	}
	return out
}

func readMemoryUsed() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	var total, available uint64
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemInfoKB(line) * 1024
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			available = parseMemInfoKB(line) * 1024
		}
	}
	if total > available {
		return total - available
	}
	return 0
}

func parseMemInfoKB(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(parts[1], 10, 64)
	return v
}

func readDiskUsage(path string) DiskMetric {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return DiskMetric{Mount: path}
	}
	total := st.Blocks * uint64(st.Bsize)
	free := st.Bavail * uint64(st.Bsize)
	used := uint64(0)
	if total > free {
		used = total - free
	}
	return DiskMetric{Mount: path, UsedBytes: used, TotalBytes: total}
}

func readNetDev() NetMetric {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return NetMetric{}
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	lineNo := 0
	var rx, tx uint64
	for s.Scan() {
		lineNo++
		if lineNo <= 2 {
			continue
		}
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(strings.ReplaceAll(line, ":", " "))
		if len(parts) < 10 {
			continue
		}
		if parts[0] == "lo" {
			continue
		}
		rxV, _ := strconv.ParseUint(parts[1], 10, 64)
		txV, _ := strconv.ParseUint(parts[9], 10, 64)
		rx += rxV
		tx += txV
	}
	return NetMetric{TXBytes: tx, RXBytes: rx}
}

func readCPUUsage() float64 {
	t1, i1, ok := readCPUStat()
	if !ok {
		return 0
	}
	time.Sleep(120 * time.Millisecond)
	t2, i2, ok := readCPUStat()
	if !ok || t2 <= t1 {
		return 0
	}
	totalDelta := float64(t2 - t1)
	idleDelta := float64(i2 - i1)
	usage := (totalDelta - idleDelta) / totalDelta
	if usage < 0 {
		return 0
	}
	if usage > 1 {
		return 1
	}
	return usage
}

func readCPUStat() (total uint64, idle uint64, ok bool) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 {
		return 0, 0, false
	}
	parts := strings.Fields(lines[0])
	if len(parts) < 5 || parts[0] != "cpu" {
		return 0, 0, false
	}
	vals := make([]uint64, 0, len(parts)-1)
	for _, p := range parts[1:] {
		v, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		vals = append(vals, v)
		total += v
	}
	idle = vals[3]
	if len(vals) > 4 {
		idle += vals[4]
	}
	return total, idle, true
}
