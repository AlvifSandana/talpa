//go:build unix
// +build unix

package safety

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func secureRemoveAll(absPath string, expected *entryIdentity) error {
	if absPath == string(filepath.Separator) {
		return errors.New("PATH_BLOCKED: root remove is forbidden")
	}
	parent := filepath.Dir(absPath)
	name := filepath.Base(absPath)

	parentFD, err := openDirNoFollow(parent)
	if err != nil {
		return err
	}
	defer unix.Close(parentFD)

	st, err := lstatAt(parentFD, name)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	}
	if expected != nil {
		current := toIdentity(st)
		if current.dev != expected.dev || current.ino != expected.ino {
			return errors.New("path changed during operation")
		}
	}
	if st.Mode&unix.S_IFMT == unix.S_IFDIR {
		mountPoints, err := loadMountPoints()
		if err != nil {
			return err
		}
		id := toIdentity(st)
		if err := removeDirChildrenByName(parentFD, name, uint64(st.Dev), absPath, absPath, mountPoints); err != nil {
			return err
		}
		if err := ensureIdentityAt(parentFD, name, id); err != nil {
			return err
		}
		if err := unix.Unlinkat(parentFD, name, unix.AT_REMOVEDIR); err != nil && !errors.Is(err, unix.ENOENT) {
			return err
		}
		return nil
	}
	if err := ensureIdentityAt(parentFD, name, toIdentity(st)); err != nil {
		return err
	}
	if err := unix.Unlinkat(parentFD, name, 0); err != nil && !errors.Is(err, unix.ENOENT) {
		return err
	}
	return nil
}

func openDirNoFollow(path string) (int, error) {
	if !filepath.IsAbs(path) {
		return -1, unix.EINVAL
	}
	if path == string(filepath.Separator) {
		return unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	}
	rootFD, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, err
	}
	cur := rootFD
	components := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	for _, c := range components {
		if c == "" || c == "." {
			continue
		}
		next, err := unix.Openat(cur, c, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
		if err != nil {
			_ = unix.Close(cur)
			return -1, err
		}
		_ = unix.Close(cur)
		cur = next
	}
	return cur, nil
}

func lstatAt(parentFD int, name string) (unix.Stat_t, error) {
	var st unix.Stat_t
	err := unix.Fstatat(parentFD, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	return st, err
}

func removeDirChildrenByName(parentFD int, name string, rootDev uint64, rootPath string, dirPath string, mountPoints map[string]struct{}) error {
	dirFD, err := unix.Openat(parentFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	var dirSt unix.Stat_t
	if err := unix.Fstat(dirFD, &dirSt); err != nil {
		_ = unix.Close(dirFD)
		return err
	}
	if uint64(dirSt.Dev) != rootDev {
		_ = unix.Close(dirFD)
		return errors.New("PATH_BLOCKED: crossing filesystem boundary")
	}
	names, err := readDirNames(dirFD)
	if err != nil {
		_ = unix.Close(dirFD)
		return err
	}
	for _, child := range names {
		childPath := filepath.Join(dirPath, child)
		st, err := lstatAt(dirFD, child)
		if err != nil {
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			_ = unix.Close(dirFD)
			return err
		}
		if st.Mode&unix.S_IFMT == unix.S_IFDIR {
			if isMountBoundaryChild(childPath, rootPath, mountPoints) {
				_ = unix.Close(dirFD)
				return errors.New("PATH_BLOCKED: crossing mount boundary")
			}
			if uint64(st.Dev) != rootDev {
				_ = unix.Close(dirFD)
				return errors.New("PATH_BLOCKED: crossing filesystem boundary")
			}
			id := toIdentity(st)
			if err := removeDirChildrenByName(dirFD, child, rootDev, rootPath, childPath, mountPoints); err != nil {
				_ = unix.Close(dirFD)
				return err
			}
			if err := ensureIdentityAt(dirFD, child, id); err != nil {
				_ = unix.Close(dirFD)
				return err
			}
			if err := unix.Unlinkat(dirFD, child, unix.AT_REMOVEDIR); err != nil && !errors.Is(err, unix.ENOENT) {
				_ = unix.Close(dirFD)
				return err
			}
			continue
		}
		if err := ensureIdentityAt(dirFD, child, toIdentity(st)); err != nil {
			_ = unix.Close(dirFD)
			return err
		}
		if err := unix.Unlinkat(dirFD, child, 0); err != nil && !errors.Is(err, unix.ENOENT) {
			_ = unix.Close(dirFD)
			return err
		}
	}
	return unix.Close(dirFD)
}

func loadMountPoints() (map[string]struct{}, error) {
	out := map[string]struct{}{}
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, errors.New("PATH_BLOCKED: mount metadata unavailable")
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 5 {
			continue
		}
		mp := decodeMountInfoPath(fields[4])
		if mp == "" {
			continue
		}
		clean := filepath.Clean(mp)
		out[clean] = struct{}{}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeMountInfoPath(raw string) string {
	r := strings.ReplaceAll(raw, "\\040", " ")
	r = strings.ReplaceAll(r, "\\011", "\t")
	r = strings.ReplaceAll(r, "\\012", "\n")
	r = strings.ReplaceAll(r, "\\134", "\\")
	return r
}

func isMountBoundaryChild(path string, rootPath string, mountPoints map[string]struct{}) bool {
	if len(mountPoints) == 0 {
		return false
	}
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(rootPath)
	if cleanPath == cleanRoot {
		return false
	}
	_, ok := mountPoints[cleanPath]
	return ok
}

func toIdentity(st unix.Stat_t) entryIdentity {
	return entryIdentity{dev: uint64(st.Dev), ino: uint64(st.Ino)}
}

func ensureIdentityAt(parentFD int, name string, want entryIdentity) error {
	cur, err := lstatAt(parentFD, name)
	if err != nil {
		return err
	}
	id := toIdentity(cur)
	if id.dev != want.dev || id.ino != want.ino {
		return errors.New("path changed during operation")
	}
	return nil
}

func readDirNames(dirFD int) ([]string, error) {
	dupFD, err := unix.Dup(dirFD)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(dupFD), "dir")
	if file == nil {
		_ = unix.Close(dupFD)
		return nil, errors.New("failed to open directory stream")
	}
	defer file.Close()
	entries, err := file.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}
