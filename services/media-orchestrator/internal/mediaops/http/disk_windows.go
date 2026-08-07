//go:build windows

package http

import "golang.org/x/sys/windows"

func diskStats(path string) (available, total uint64) {
	root, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var freeForCaller, totalBytes, freeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(root, &freeForCaller, &totalBytes, &freeBytes); err != nil {
		return 0, 0
	}
	return freeForCaller, totalBytes
}

