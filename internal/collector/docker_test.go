package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContainersJSON(t *testing.T) {
	data := []byte(`[
		{
			"Id": "6fa772d1595b1d127d31b515c68c546fba8394cb8e96588c79dfb7f9447d7bd8",
			"Names": ["/pmtoptest"],
			"Image": "nginx:alpine",
			"State": "running",
			"Status": "Up 3 seconds",
			"Ports": [
				{"IP": "0.0.0.0", "PrivatePort": 80, "PublicPort": 8088, "Type": "tcp"}
			]
		},
		{
			"Id": "abcdef1234567890",
			"Names": ["/other"],
			"Image": "redis:7",
			"State": "exited",
			"Status": "Exited (0) 5 seconds ago",
			"Ports": []
		}
	]`)
	m := ParseContainersJSON(data)
	require.Len(t, m, 2)

	c, ok := m["6fa772d1595b1d127d31b515c68c546fba8394cb8e96588c79dfb7f9447d7bd8"]
	require.True(t, ok)
	assert.Equal(t, "pmtoptest", c.Name)
	assert.Equal(t, "nginx:alpine", c.Image)
	assert.Equal(t, "running", c.State)
	assert.Equal(t, "Up 3 seconds", c.Status)
	require.Len(t, c.Ports, 1)
	assert.Equal(t, uint16(80), c.Ports[0].PrivatePort)
	assert.Equal(t, uint16(8088), c.Ports[0].PublicPort)
	assert.Equal(t, "tcp", c.Ports[0].Type)

	c, ok = m["abcdef1234567890"]
	require.True(t, ok)
	assert.Equal(t, "other", c.Name)
	assert.Empty(t, c.Ports)
}

func TestParseContainersJSON_Invalid(t *testing.T) {
	assert.Nil(t, ParseContainersJSON([]byte("not json")))
	assert.Nil(t, ParseContainersJSON(nil))
}

func TestNoResolver(t *testing.T) {
	var nr noResolver
	_, ok := nr.Resolve("anything")
	assert.False(t, ok)
	assert.Nil(t, nr.All())
}
