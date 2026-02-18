//go:build !unix
// +build !unix

package analyze

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var renameAt = func(_ int, _ string, _ int, _ string) error { return nil }

func secureMoveToTrash(srcPath, trashAbs string, allowedRoots []string, whitelist []string, expectedDev uint64, expectedIno uint64) error {
	_ = allowedRoots
	_ = whitelist
	_ = expectedDev
	_ = expectedIno
	base := filepath.Base(srcPath)
	if base == "." || base == string(filepath.Separator) {
		return fmt.Errorf("invalid trash source path: %s", srcPath)
	}
	for i := 0; i < 32; i++ {
		suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
		if i > 0 {
			suffix = suffix + "-" + strconv.Itoa(i)
		}
		dst := filepath.Join(trashAbs, base+"-"+suffix)
		err := os.Rename(srcPath, dst)
		if err == nil {
			return nil
		}
		if os.IsExist(err) {
			continue
		}
		return err
	}
	return fmt.Errorf("failed to reserve unique trash destination for %s", srcPath)
}
