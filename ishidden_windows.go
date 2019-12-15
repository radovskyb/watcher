// +build windows

package watcher

import (
	"os"
	"syscall"
)

func isHiddenFile(path string) (bool, error) {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = &os.PathError{
				Op:   "isHidden",
				Path: path,
				Err:  err,
			}
		}
		return false, err
	}

	attributes, err := syscall.GetFileAttributes(pointer)
	if err != nil {
		if os.IsNotExist(err) {
			err = &os.PathError{
				Op:   "isHidden",
				Path: path,
				Err:  err,
			}
		}
		return false, err
	}

	return attributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0, nil
}
