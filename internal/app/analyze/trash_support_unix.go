//go:build unix
// +build unix

package analyze

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func ensureTrashActionSupported() error {
	return nil
}

func prepareTrashDir(path string) error {
	if err := ensureDirPathNoFollow(path, 0o700); err != nil {
		return err
	}
	fd, err := openDirNoFollow(path)
	if err != nil {
		return err
	}
	_ = unix.Close(fd)
	return nil
}

func ensureDirPathNoFollow(path string, perm uint32) error {
	if !filepath.IsAbs(path) {
		return unix.EINVAL
	}
	if path == string(filepath.Separator) {
		return nil
	}
	rootFD, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	cur := rootFD
	components := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	for _, c := range components {
		if c == "" || c == "." {
			continue
		}
		next, err := unix.Openat(cur, c, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
		if err != nil {
			if err == unix.ENOENT {
				if mkErr := unix.Mkdirat(cur, c, perm); mkErr != nil && mkErr != unix.EEXIST {
					_ = unix.Close(cur)
					return mkErr
				}
				next, err = unix.Openat(cur, c, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
			}
			if err != nil {
				_ = unix.Close(cur)
				return err
			}
		}
		_ = unix.Close(cur)
		cur = next
	}
	_ = unix.Close(cur)
	return nil
}
