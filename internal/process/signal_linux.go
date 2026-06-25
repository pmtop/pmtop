//go:build linux

package process

import "syscall"

// SendSignal delivers sig to pid via syscall.Kill (Linux).
func SendSignal(pid int, sig Signal) error {
	if err := ValidatePID(pid); err != nil {
		return err
	}
	return syscall.Kill(pid, syscall.Signal(sig.Num))
}
