//go:build windows

package cli

func isTerminalFD(fd uintptr) bool {
	return false
}
