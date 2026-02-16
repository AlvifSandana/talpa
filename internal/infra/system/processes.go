package system

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var out []ProcessStat
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		cmd := readCmdline(pid)
		rss := readRSS(pid)
		out = append(out, ProcessStat{PID: pid, Command: cmd, MemBytes: rss, CPUPercent: 0})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].MemBytes > out[j].MemBytes })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
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
	return 0
}
