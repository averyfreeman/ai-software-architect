//go:build windows

package architect

import (
	"os"
	"syscall"
)

func isReparsePoint(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	wide, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attributes, err := syscall.GetFileAttributes(wide)
	return err == nil && attributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
