//go:build unix
// +build unix

package analyze

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

var renameAt = unix.Renameat

type entryIdentity struct {
	dev uint64
	ino uint64
}

func secureMoveToTrash(srcPath, trashAbs string, allowedRoots []string, whitelist []string, expectedDev uint64, expectedIno uint64) error {
	if !isAllowedTrashSource(srcPath, allowedRoots, whitelist) {
		return errors.New("PATH_BLOCKED: outside allowed roots")
	}
	if strings.TrimSpace(srcPath) == "" {
		return errors.New("PATH_INVALID: empty source path")
	}
	srcParent := filepath.Dir(srcPath)
	srcName := filepath.Base(srcPath)

	srcParentFD, err := openDirNoFollow(srcParent)
	if err != nil {
		return err
	}
	defer unix.Close(srcParentFD)

	trashFD, err := openDirNoFollow(trashAbs)
	if err != nil {
		return err
	}
	defer unix.Close(trashFD)
	containerName, err := reserveTrashContainer(trashFD, srcName)
	if err != nil {
		return err
	}
	containerFD, err := openDirAtNoFollow(trashFD, containerName)
	if err != nil {
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}

	const payloadName = "item"

	if err := ensureIdentityAt(srcParentFD, srcName, resolveExpectedIdentity(srcPath, expectedDev, expectedIno)); err != nil {
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}
	if err := renameAt(srcParentFD, srcName, containerFD, payloadName); err == nil {
		_ = unix.Close(containerFD)
		return nil
	} else if !errors.Is(err, unix.EXDEV) {
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}

	srcSt, err := lstatAt(srcParentFD, srcName)
	if err != nil {
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}
	want := resolveExpectedIdentity(srcPath, expectedDev, expectedIno)
	if err := ensureIdentityAt(srcParentFD, srcName, want); err != nil {
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}
	srcID := toIdentity(srcSt)
	mountPoints, err := loadMountPoints()
	if err != nil {
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}
	rootPath := filepath.Clean(srcPath)

	if err := copyEntryAt(srcParentFD, srcName, containerFD, payloadName, uint64(srcSt.Dev), rootPath, rootPath, mountPoints); err != nil {
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}
	if err := ensureIdentityAt(srcParentFD, srcName, srcID); err != nil {
		_ = removeEntryAt(containerFD, payloadName)
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}
	if err := removeEntryAt(srcParentFD, srcName); err != nil {
		_ = removeEntryAt(containerFD, payloadName)
		_ = unix.Close(containerFD)
		_ = unix.Unlinkat(trashFD, containerName, unix.AT_REMOVEDIR)
		return err
	}
	_ = unix.Close(containerFD)
	return nil
}

func reserveTrashContainer(trashFD int, srcBase string) (string, error) {
	base := srcBase + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	for i := 0; i < 32; i++ {
		name := base
		if i > 0 {
			name = base + "-" + strconv.Itoa(i)
		}
		if err := unix.Mkdirat(trashFD, name, 0o700); err == nil {
			return name, nil
		} else if errors.Is(err, unix.EEXIST) {
			continue
		} else {
			return "", err
		}
	}
	return "", errors.New("failed to reserve unique trash container")
}

func openDirNoFollow(path string) (int, error) {
	if !filepath.IsAbs(path) {
		return -1, errors.New("path must be absolute")
	}
	if path == string(filepath.Separator) {
		return unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	}
	rootFD, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, err
	}
	components := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	cur := rootFD
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

func openDirAtNoFollow(parentFD int, name string) (int, error) {
	return unix.Openat(parentFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
}

func lstatAt(parentFD int, name string) (unix.Stat_t, error) {
	var st unix.Stat_t
	err := unix.Fstatat(parentFD, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	return st, err
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

func copyEntryAt(srcParentFD int, srcName string, dstParentFD int, dstName string, rootDev uint64, rootPath string, curPath string, mountPoints map[string]struct{}) error {
	st, err := lstatAt(srcParentFD, srcName)
	if err != nil {
		return err
	}
	mode := st.Mode & unix.S_IFMT
	switch mode {
	case unix.S_IFDIR:
		if isMountBoundaryChild(curPath, rootPath, mountPoints) {
			return errors.New("PATH_BLOCKED: crossing mount boundary")
		}
		if uint64(st.Dev) != rootDev {
			return errors.New("PATH_BLOCKED: crossing filesystem boundary")
		}
		perm := uint32(st.Mode & 0o777)
		if err := unix.Mkdirat(dstParentFD, dstName, perm); err != nil {
			return err
		}
		srcDirFD, err := openDirAtNoFollow(srcParentFD, srcName)
		if err != nil {
			return err
		}
		defer unix.Close(srcDirFD)
		dstDirFD, err := openDirAtNoFollow(dstParentFD, dstName)
		if err != nil {
			return err
		}
		defer unix.Close(dstDirFD)
		names, err := readDirNames(srcDirFD)
		if err != nil {
			return err
		}
		for _, child := range names {
			childPath := filepath.Join(curPath, child)
			if err := copyEntryAt(srcDirFD, child, dstDirFD, child, rootDev, rootPath, childPath, mountPoints); err != nil {
				return err
			}
		}
		return nil
	case unix.S_IFLNK:
		target, err := readlinkAt(srcParentFD, srcName)
		if err != nil {
			return err
		}
		return unix.Symlinkat(target, dstParentFD, dstName)
	case unix.S_IFREG:
		return copyFileAt(srcParentFD, srcName, dstParentFD, dstName, uint32(st.Mode&0o777))
	default:
		return fmt.Errorf("unsupported file type for trash fallback: mode=%o", mode)
	}
}

func copyFileAt(srcParentFD int, srcName string, dstParentFD int, dstName string, perm uint32) error {
	srcFD, err := unix.Openat(srcParentFD, srcName, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	dstFD, err := unix.Openat(dstParentFD, dstName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW, perm)
	if err != nil {
		_ = unix.Close(srcFD)
		return err
	}

	srcFile := os.NewFile(uintptr(srcFD), "src")
	dstFile := os.NewFile(uintptr(dstFD), "dst")
	if srcFile == nil || dstFile == nil {
		if srcFile != nil {
			_ = srcFile.Close()
		} else {
			_ = unix.Close(srcFD)
		}
		if dstFile != nil {
			_ = dstFile.Close()
		} else {
			_ = unix.Close(dstFD)
		}
		return errors.New("failed to bind file descriptor")
	}
	errCopy := copyAndCloseFiles(srcFile, dstFile)
	if errCopy != nil {
		_ = unix.Unlinkat(dstParentFD, dstName, 0)
	}
	return errCopy
}

func copyAndCloseFiles(srcFile, dstFile *os.File) error {
	defer srcFile.Close()
	defer dstFile.Close()
	_, err := io.Copy(dstFile, srcFile)
	return err
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

func readlinkAt(parentFD int, name string) (string, error) {
	buf := make([]byte, 256)
	for {
		n, err := unix.Readlinkat(parentFD, name, buf)
		if err != nil {
			return "", err
		}
		if n < len(buf) {
			return string(buf[:n]), nil
		}
		buf = make([]byte, len(buf)*2)
	}
}

func removeEntryAt(parentFD int, name string) error {
	st, err := lstatAt(parentFD, name)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	}
	if st.Mode&unix.S_IFMT == unix.S_IFDIR {
		mountPoints, err := loadMountPoints()
		if err != nil {
			return err
		}
		entryPath := filepath.Clean(filepath.Join(dirPathFromFD(parentFD), name))
		id := toIdentity(st)
		if err := removeDirChildrenByName(parentFD, name, uint64(st.Dev), entryPath, entryPath, mountPoints); err != nil {
			return err
		}
		if err := ensureIdentityAt(parentFD, name, id); err != nil {
			return err
		}
		return unix.Unlinkat(parentFD, name, unix.AT_REMOVEDIR)
	}
	if err := ensureIdentityAt(parentFD, name, toIdentity(st)); err != nil {
		return err
	}
	return unix.Unlinkat(parentFD, name, 0)
}

func removeDirChildrenByName(parentFD int, name string, rootDev uint64, rootPath string, dirPath string, mountPoints map[string]struct{}) error {
	dirFD, err := openDirAtNoFollow(parentFD, name)
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
		unix.Close(dirFD)
		return err
	}
	for _, child := range names {
		childPath := filepath.Join(dirPath, child)
		st, err := lstatAt(dirFD, child)
		if err != nil {
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			unix.Close(dirFD)
			return err
		}
		if st.Mode&unix.S_IFMT == unix.S_IFDIR {
			if isMountBoundaryChild(childPath, rootPath, mountPoints) {
				unix.Close(dirFD)
				return errors.New("PATH_BLOCKED: crossing mount boundary")
			}
			if uint64(st.Dev) != rootDev {
				unix.Close(dirFD)
				return errors.New("PATH_BLOCKED: crossing filesystem boundary")
			}
			id := toIdentity(st)
			if err := removeDirChildrenByName(dirFD, child, rootDev, rootPath, childPath, mountPoints); err != nil {
				unix.Close(dirFD)
				return err
			}
			if err := ensureIdentityAt(dirFD, child, id); err != nil {
				unix.Close(dirFD)
				return err
			}
			if err := unix.Unlinkat(dirFD, child, unix.AT_REMOVEDIR); err != nil && !errors.Is(err, unix.ENOENT) {
				unix.Close(dirFD)
				return err
			}
			continue
		}
		if err := ensureIdentityAt(dirFD, child, toIdentity(st)); err != nil {
			unix.Close(dirFD)
			return err
		}
		if err := unix.Unlinkat(dirFD, child, 0); err != nil && !errors.Is(err, unix.ENOENT) {
			unix.Close(dirFD)
			return err
		}
	}
	if err := unix.Close(dirFD); err != nil {
		return err
	}
	return nil
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
		out[filepath.Clean(mp)] = struct{}{}
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

func resolveExpectedIdentity(path string, expectedDev uint64, expectedIno uint64) entryIdentity {
	if expectedDev != 0 || expectedIno != 0 {
		return entryIdentity{dev: expectedDev, ino: expectedIno}
	}
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return entryIdentity{}
	}
	return entryIdentity{dev: uint64(st.Dev), ino: uint64(st.Ino)}
}

func dirPathFromFD(fd int) string {
	link := "/proc/self/fd/" + strconv.Itoa(fd)
	target, err := os.Readlink(link)
	if err != nil {
		return string(filepath.Separator)
	}
	return filepath.Clean(target)
}
