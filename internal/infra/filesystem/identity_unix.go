//go:build unix
// +build unix

package filesystem

import (
	"os"
	"syscall"
)

func statIdentity(fi os.FileInfo) (uint64, uint64) {
	if fi == nil || fi.Sys() == nil {
		return 0, 0
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0
	}
	return uint64(st.Dev), uint64(st.Ino)
}
