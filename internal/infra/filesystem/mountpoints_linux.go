//go:build linux
// +build linux

package filesystem

import "os"

func readMountPoints() (map[string]string, error) {
	b, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	return parseMountInfo(string(b)), nil
}
