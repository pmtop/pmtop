package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSignal(t *testing.T) {
	s := DefaultSignal()
	assert.Equal(t, "SIGTERM", s.Name)
	assert.Equal(t, 15, s.Num)
}

func TestParseSignal(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"SIGTERM", "SIGTERM", true},
		{"sigterm", "SIGTERM", true},
		{"15", "SIGTERM", true},
		{"9", "SIGKILL", true},
		{"SIGHUP", "SIGHUP", true},
		{"  sigint  ", "SIGINT", true},
		{"SIGBOGUS", "", false},
		{"99", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := ParseSignal(c.in)
		assert.Equal(t, c.ok, ok, c.in)
		if ok {
			assert.Equal(t, c.want, got.Name, c.in)
		}
	}
}

func TestSignalsContainsAll(t *testing.T) {
	names := map[string]bool{}
	for _, s := range Signals {
		names[s.Name] = true
	}
	for _, n := range []string{"SIGHUP", "SIGINT", "SIGTERM", "SIGKILL", "SIGUSR1", "SIGUSR2"} {
		assert.True(t, names[n], "missing %s", n)
	}
}

func TestValidatePID(t *testing.T) {
	assert.NoError(t, ValidatePID(1))
	assert.Error(t, ValidatePID(0))
	assert.Error(t, ValidatePID(-1))
}
