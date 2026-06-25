//go:build linux

package process

import (
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendSignal_TerminatesChild(t *testing.T) {
	// Spawn a local sleep subprocess (a test fixture, not an external service)
	// and confirm SIGTERM delivered via SendSignal ends it.
	cmd := exec.Command("sleep", "30")
	require.NoError(t, cmd.Start())
	pid := cmd.Process.Pid
	require.Greater(t, pid, 0)

	require.NoError(t, SendSignal(pid, Signal{Name: "SIGTERM", Num: 15}))

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		// Process was terminated by a signal.
		assert.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("child was not terminated by SIGTERM")
	}
}

func TestSendSignal_InvalidPID(t *testing.T) {
	assert.Error(t, SendSignal(0, DefaultSignal()))
	assert.Error(t, SendSignal(-1, DefaultSignal()))
}
