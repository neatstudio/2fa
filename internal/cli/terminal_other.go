//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !windows

package cli

func isTerminalFD(fd uintptr) bool {
	return false
}
