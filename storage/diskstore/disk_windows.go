// +build windows

package diskstore

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var errSharingViolation = syscall.Errno(32)

func isBusy(err error) bool {
	err = underlyingError(err)
	return err == errSharingViolation
}

func diskInfoFromPath(path string) (filesytemId string, amount int64, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	amount, err = getDiskFreeSpace(absPath)
	if err != nil {
		return "", -1, err
	}

	filesytemId, err = getVolumeSerialNumber(absPath)
	if err != nil {
		return "", amount, err
	}

	return filesytemId, amount, nil
}

var (
	kernel32             = syscall.MustLoadDLL("kernel32.dll")
	procGetDiskFreeSpace = kernel32.MustFindProc("GetDiskFreeSpaceExW")
)

func getDiskFreeSpace(path string) (int64, error) {
	path16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return -1, err
	}

	var bytes int64
	_, _, err = procGetDiskFreeSpace.Call(uintptr(unsafe.Pointer(path16)), uintptr(unsafe.Pointer(&bytes)), 0, 0)
	err = ignoreSuccess(err)
	return bytes, err
}

func getVolumeSerialNumber(path string) (string, error) {
	path16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", nil
	}

	var volumePath [1024]uint16
	err = windows.GetVolumePathName(path16, &volumePath[0], uint32(len(volumePath)))
	if err != nil {
		return "", err
	}

	var volumeSerial uint32

	err = windows.GetVolumeInformation(
		&volumePath[0],
		nil, 0, // volume name buffer
		&volumeSerial,
		nil,    // maximum component length
		nil,    // filesystem flags
		nil, 0, // filesystem name buffer
	)
	err = ignoreSuccess(err)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%08x", volumeSerial), nil
}

// windows api occasionally returns
func ignoreSuccess(err error) error {
	if err == syscall.Errno(0) {
		return nil
	}
	return err
}
