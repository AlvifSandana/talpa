//go:build !linux
// +build !linux

package filesystem

func readMountPoints() (map[string]string, error) {
	return map[string]string{}, nil
}
