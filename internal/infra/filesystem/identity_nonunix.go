//go:build !unix
// +build !unix

package filesystem

import "os"

func statIdentity(fi os.FileInfo) (uint64, uint64) {
	_ = fi
	return 0, 0
}
