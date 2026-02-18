//go:build !unix
// +build !unix

package analyze

import "os"

func ensureTrashActionSupported() error {
	return nil
}

func prepareTrashDir(path string) error {
	if path == "" {
		return nil
	}
	err := os.MkdirAll(path, 0o700)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}
