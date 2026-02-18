//go:build !unix
// +build !unix

package safety

import "os"

func secureRemoveAll(absPath string, expected *entryIdentity) error {
	_ = expected
	err := os.RemoveAll(absPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
