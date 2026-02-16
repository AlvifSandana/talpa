package system

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ProcessStat struct {
	PID        int     `json:"pid"`
	Command    string  `json:"command"`
	CPUPercent float64 `json:"cpu_percent"`
	MemBytes   uint64  `json:"mem_bytes"`
}

func TopProcesses(limit int) []ProcessStat {
	if limit <= 0 {
		limit = 5
	}

	snap1 := readSnapshot()
	time.Sleep(120 * time.Millisecond)
	snap2 := readSnapshot()

	var out []ProcessStat
	deltaTotal := uint64(0)
	if snap2.TotalCPU > snap1.TotalCPU {
		deltaTotal = snap2.TotalCPU - snap1.TotalCPU
	}

	for _, s2 := range snap2.Procs {
		cpuPercent := 0.0
		if deltaTotal > 0 {
			if s1, ok := snap1.Procs[s2.PID]; ok && s2.CPUJiffies >= s1.CPUJiffies {
				deltaProc := s2.CPUJiffies - s1.CPUJiffies
				cpuPercent = computeCPUPercent(deltaProc, deltaTotal)
			}
		}
		out = append(out, ProcessStat{PID: s2.PID, Command: s2.Command, MemBytes: s2.MemBytes, CPUPercent: cpuPercent})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].MemBytes > out[j].MemBytes })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

type procSample struct {
	PID        int
	Command    string
	MemBytes   uint64
	CPUJiffies uint64
}

type snapshot struct {
	TotalCPU uint64
	Procs    map[int]procSample
}

func readSnapshot() snapshot {
	total, _ := readTotalCPUJiffies()
	procs := readProcSnapshot()
	return snapshot{TotalCPU: total, Procs: procs}
}

func readProcSnapshot() map[int]procSample {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	out := make(map[int]procSample, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		cpu, ok := readProcCPUJiffies(pid)
		if !ok {
			continue
		}
		out[pid] = procSample{
			PID:        pid,
			Command:    readCmdline(pid),
			MemBytes:   readRSS(pid),
			CPUJiffies: cpu,
		}
	}
	return out
}

func readTotalCPUJiffies() (uint64, bool) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, false
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 {
		return 0, false
	}
	return parseTotalCPUJiffies(lines[0])
}

func parseTotalCPUJiffies(line string) (uint64, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 || parts[0] != "cpu" {
		return 0, false
	}
	var total uint64
	for _, p := range parts[1:] {
		v, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return 0, false
		}
		total += v
	}
	return total, true
}

func readProcCPUJiffies(pid int) (uint64, bool) {
	b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, false
	}
	return parseProcCPUJiffies(string(b))
}

func parseProcCPUJiffies(line string) (uint64, bool) {
	closeIdx := strings.LastIndex(line, ")")
	if closeIdx == -1 || closeIdx+2 >= len(line) {
		return 0, false
	}
	if line[closeIdx+1] != ' ' {
		return 0, false
	}
	tail := strings.Fields(line[closeIdx+2:])
	if len(tail) < 13 {
		return 0, false
	}
	utime, err := strconv.ParseUint(tail[11], 10, 64)
	if err != nil {
		return 0, false
	}
	stime, err := strconv.ParseUint(tail[12], 10, 64)
	if err != nil {
		return 0, false
	}
	return utime + stime, true
}

func readCmdline(pid int) string {
	b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil || len(b) == 0 {
		return "unknown"
	}
	parts := strings.Split(string(b), "\x00")
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			return p
		}
	}
	return "unknown"
}

func readRSS(pid int) uint64 {
	f, err := os.Open(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				v, _ := strconv.ParseUint(parts[1], 10, 64)
				return v * 1024
			}
		}
	}
	if s.Err() != nil {
		return 0
	}
	return 0
}

func computeCPUPercent(deltaProc uint64, deltaTotal uint64) float64 {
	if deltaTotal == 0 {
		return 0
	}
	percent := (float64(deltaProc) / float64(deltaTotal)) * 100
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}
