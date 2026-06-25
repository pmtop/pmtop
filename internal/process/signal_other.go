//go:build !linux

package process

import "errors"

// SendSignal is a no-op stub on non-Linux platforms (no signals / no /proc).
func SendSignal(pid int, sig Signal) error {
	return errors.New("signal sending requires Linux")
}
