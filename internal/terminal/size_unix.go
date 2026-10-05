//go:build darwin || linux

package terminal

import (
	"fmt"
	"syscall"
	"unsafe"
)

type windowSize struct {
	Rows    uint16
	Columns uint16
	X       uint16
	Y       uint16
}

// Size returns the current terminal dimensions for fd.
func Size(fd uintptr) (rows, columns int, err error) {
	var size windowSize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&size)))
	if errno != 0 {
		return 0, 0, fmt.Errorf("read terminal size: %w", errno)
	}
	if size.Rows == 0 || size.Columns == 0 {
		return 0, 0, fmt.Errorf("read terminal size: empty dimensions")
	}
	return int(size.Rows), int(size.Columns), nil
}
