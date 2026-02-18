package status

import (
	"bufio"
	"context"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"talpa/internal/app/common"
	"talpa/internal/domain/model"
	"talpa/internal/infra/system"
)

type Service struct {
	readers statusReaders
}

type statusReaders struct {
	loadAvg    func() [3]float64
	memory     func() MemoryMetric
	diskUsage  func(string) DiskMetric
	diskUsageN func(int) []DiskMetric
	throughput func() (float64, DiskIOMetric, NetMetric)
	diskIO     func() DiskIOMetric
	net        func() NetMetric
	cpuUsage   func() float64
	ipAddrs    func() []string
	topProcess func(int) []system.ProcessStat
}

type Metrics struct {
	CPUUsage         float64      `json:"cpu_usage"`
	LoadAvg          [3]float64   `json:"load_avg"`
	MemoryTotalBytes uint64       `json:"memory_total_bytes"`
	MemoryUsedBytes  uint64       `json:"memory_used_bytes"`
	SwapUsedBytes    uint64       `json:"swap_used_bytes"`
	SwapTotalBytes   uint64       `json:"swap_total_bytes"`
	DiskUsage        []DiskMetric `json:"disk_usage"`
	DiskIO           DiskIOMetric `json:"disk_io"`
	Net              NetMetric    `json:"net"`
	IPAddresses      []string     `json:"ip_addresses"`
	TopProcesses     []Process    `json:"top_processes"`
}

type MemoryMetric struct {
	UsedBytes  uint64
	TotalBytes uint64
	SwapUsed   uint64
	SwapTotal  uint64
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
	TXBPS   uint64 `json:"tx_bps"`
	RXBPS   uint64 `json:"rx_bps"`
}

type DiskIOMetric struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadBPS    uint64 `json:"read_bps"`
	WriteBPS   uint64 `json:"write_bps"`
}

func defaultStatusReaders() statusReaders {
	return statusReaders{
		loadAvg:    readLoadAvg,
		memory:     readMemoryMetric,
		diskUsage:  readDiskUsage,
		diskUsageN: readTopDiskUsage,
		throughput: readThroughputSnapshot,
		diskIO:     readDiskIO,
		net:        readNetDev,
		cpuUsage:   readCPUUsage,
		ipAddrs:    readIPAddresses,
		topProcess: system.TopProcesses,
	}
}

func NewService() Service { return Service{readers: defaultStatusReaders()} }

func (s Service) Run(ctx context.Context, app *common.AppContext) (model.CommandResult, error) {
	_ = ctx
	start := time.Now()
	r := defaultStatusReaders()
	if s.readers.loadAvg != nil {
		r.loadAvg = s.readers.loadAvg
	}
	if s.readers.memory != nil {
		r.memory = s.readers.memory
	}
	if s.readers.diskUsage != nil {
		r.diskUsage = s.readers.diskUsage
	}
	if s.readers.diskUsageN != nil {
		r.diskUsageN = s.readers.diskUsageN
	}
	if s.readers.throughput != nil {
		r.throughput = s.readers.throughput
	}
	if s.readers.diskIO != nil {
		r.diskIO = s.readers.diskIO
	}
	if s.readers.net != nil {
		r.net = s.readers.net
	}
	if s.readers.cpuUsage != nil {
		r.cpuUsage = s.readers.cpuUsage
	}
	if s.readers.ipAddrs != nil {
		r.ipAddrs = s.readers.ipAddrs
	}
	if s.readers.topProcess != nil {
		r.topProcess = s.readers.topProcess
	}

	load := r.loadAvg()
	mem := r.memory()
	disk := make([]DiskMetric, 0, 3)
	if s.readers.diskUsage != nil && s.readers.diskUsageN == nil {
		disk = []DiskMetric{r.diskUsage("/")}
	} else {
		disk = r.diskUsageN(3)
		if len(disk) == 0 {
			disk = []DiskMetric{r.diskUsage("/")}
		}
	}
	useThroughput := s.readers.throughput != nil || (s.readers.cpuUsage == nil && s.readers.diskIO == nil && s.readers.net == nil)
	diskIO := DiskIOMetric{}
	net := NetMetric{}
	cpuUsage := 0.0
	if useThroughput && r.throughput != nil {
		cpuUsage, diskIO, net = r.throughput()
	} else {
		diskIO = r.diskIO()
		net = r.net()
		cpuUsage = r.cpuUsage()
	}
	ipAddrs := r.ipAddrs()

	top := r.topProcess(app.Options.StatusTop)
	procs := make([]Process, 0, len(top))
	for _, p := range top {
		procs = append(procs, Process{PID: p.PID, Command: p.Command, CPUPercent: p.CPUPercent, MemBytes: p.MemBytes})
	}

	metrics := Metrics{
		CPUUsage:         cpuUsage,
		LoadAvg:          load,
		MemoryTotalBytes: mem.TotalBytes,
		MemoryUsedBytes:  mem.UsedBytes,
		SwapUsedBytes:    mem.SwapUsed,
		SwapTotalBytes:   mem.SwapTotal,
		DiskUsage:        disk,
		DiskIO:           diskIO,
		Net:              net,
		IPAddresses:      ipAddrs,
		TopProcesses:     procs,
	}

	return model.CommandResult{
		SchemaVersion: "1.0",
		Command:       "status",
		Timestamp:     time.Now().UTC(),
		DurationMS:    time.Since(start).Milliseconds(),
		Metrics:       metrics,
	}, nil
}

func readThroughputSnapshot() (float64, DiskIOMetric, NetMetric) {
	t1, i1, okCPU1 := readCPUStat()
	rx1, tx1 := readNetCounters()
	r1, w1 := readDiskIOCounters()
	time.Sleep(120 * time.Millisecond)
	t2, i2, okCPU2 := readCPUStat()
	rx2, tx2 := readNetCounters()
	r2, w2 := readDiskIOCounters()

	cpu := 0.0
	if okCPU1 && okCPU2 && t2 > t1 {
		totalDelta := float64(t2 - t1)
		idleDelta := float64(i2 - i1)
		usage := (totalDelta - idleDelta) / totalDelta
		if usage < 0 {
			usage = 0
		}
		if usage > 1 {
			usage = 1
		}
		cpu = usage
	}

	deltaRX := uint64(0)
	deltaTX := uint64(0)
	if rx2 > rx1 {
		deltaRX = rx2 - rx1
	}
	if tx2 > tx1 {
		deltaTX = tx2 - tx1
	}
	net := NetMetric{TXBytes: tx2, RXBytes: rx2, TXBPS: deltaTX * 1000 / 120, RXBPS: deltaRX * 1000 / 120}

	deltaR := uint64(0)
	deltaW := uint64(0)
	if r2 > r1 {
		deltaR = r2 - r1
	}
	if w2 > w1 {
		deltaW = w2 - w1
	}
	disk := DiskIOMetric{ReadBytes: r2, WriteBytes: w2, ReadBPS: deltaR * 1000 / 120, WriteBPS: deltaW * 1000 / 120}

	return cpu, disk, net
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

func readMemoryMetric() MemoryMetric {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryMetric{}
	}
	defer f.Close()

	var total, available, swapTotal, swapFree uint64
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemInfoKB(line) * 1024
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			available = parseMemInfoKB(line) * 1024
		}
		if strings.HasPrefix(line, "SwapTotal:") {
			swapTotal = parseMemInfoKB(line) * 1024
		}
		if strings.HasPrefix(line, "SwapFree:") {
			swapFree = parseMemInfoKB(line) * 1024
		}
	}
	used := uint64(0)
	if total > available {
		used = total - available
	}
	swapUsed := uint64(0)
	if swapTotal > swapFree {
		swapUsed = swapTotal - swapFree
	}
	return MemoryMetric{UsedBytes: used, TotalBytes: total, SwapUsed: swapUsed, SwapTotal: swapTotal}
}

func readTopDiskUsage(limit int) []DiskMetric {
	if limit <= 0 {
		limit = 1
	}
	b, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil
	}
	lines := strings.Split(string(b), "\n")
	out := make([]DiskMetric, 0, limit)
	seen := map[string]struct{}{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mount := fields[1]
		fstype := fields[2]
		if fstype == "proc" || fstype == "sysfs" || fstype == "devtmpfs" || fstype == "devpts" || fstype == "tmpfs" || fstype == "cgroup2" || fstype == "overlay" {
			continue
		}
		if _, ok := seen[mount]; ok {
			continue
		}
		seen[mount] = struct{}{}
		out = append(out, readDiskUsage(mount))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UsedBytes > out[j].UsedBytes
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
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
	rx1, tx1 := readNetCounters()
	time.Sleep(120 * time.Millisecond)
	rx2, tx2 := readNetCounters()
	deltaRX := uint64(0)
	deltaTX := uint64(0)
	if rx2 > rx1 {
		deltaRX = rx2 - rx1
	}
	if tx2 > tx1 {
		deltaTX = tx2 - tx1
	}
	bpsRX := deltaRX * 1000 / 120
	bpsTX := deltaTX * 1000 / 120
	return NetMetric{TXBytes: tx2, RXBytes: rx2, TXBPS: bpsTX, RXBPS: bpsRX}
}

func readNetCounters() (uint64, uint64) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
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
	return rx, tx
}

func readDiskIO() DiskIOMetric {
	r1, w1 := readDiskIOCounters()
	time.Sleep(120 * time.Millisecond)
	r2, w2 := readDiskIOCounters()
	deltaR := uint64(0)
	deltaW := uint64(0)
	if r2 > r1 {
		deltaR = r2 - r1
	}
	if w2 > w1 {
		deltaW = w2 - w1
	}
	return DiskIOMetric{
		ReadBytes:  r2,
		WriteBytes: w2,
		ReadBPS:    deltaR * 1000 / 120,
		WriteBPS:   deltaW * 1000 / 120,
	}
}

func readDiskIOCounters() (uint64, uint64) {
	b, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return 0, 0
	}
	var readSectors uint64
	var writtenSectors uint64
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}
		name := fields[2]
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		r, errR := strconv.ParseUint(fields[5], 10, 64)
		w, errW := strconv.ParseUint(fields[9], 10, 64)
		if errR != nil || errW != nil {
			continue
		}
		readSectors += r
		writtenSectors += w
	}
	return readSectors * 512, writtenSectors * 512
}

func readIPAddresses() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]string, 0, 4)
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip == nil {
				continue
			}
			out = append(out, ip.String())
		}
	}
	sort.Strings(out)
	if len(out) > 4 {
		out = out[:4]
	}
	return out
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
