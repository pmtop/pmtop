package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDpkgS(t *testing.T) {
	pkg, ok := ParseDpkgS("nginx: /usr/sbin/nginx\n")
	assert.True(t, ok)
	assert.Equal(t, "nginx", pkg)

	// Multiple comma-separated owners -> first.
	pkg, ok = ParseDpkgS("nginx-common, nginx-core: /usr/sbin/nginx")
	assert.True(t, ok)
	assert.Equal(t, "nginx-common", pkg)

	// No owner.
	_, ok = ParseDpkgS("dpkg-query: no path found matching pattern /foo")
	assert.False(t, ok)
	_, ok = ParseDpkgS("")
	assert.False(t, ok)
}

func TestParseRpmQf(t *testing.T) {
	name, ver, ok := ParseRpmQf("nginx 1.26.0-1.el9")
	assert.True(t, ok)
	assert.Equal(t, "nginx", name)
	assert.Equal(t, "1.26.0-1.el9", ver)

	// Name only.
	name, ver, ok = ParseRpmQf("nginx")
	assert.True(t, ok)
	assert.Equal(t, "nginx", name)
	assert.Empty(t, ver)

	// Not installed.
	_, _, ok = ParseRpmQf("file /usr/sbin/nginx is not installed")
	assert.False(t, ok)
	_, _, ok = ParseRpmQf("")
	assert.False(t, ok)
}

// TestPackageOwner_Executes exercises the dpkg/rpm exec paths without asserting
// a specific owner (package managers may be absent). It must not panic and must
// return ErrNoPackage when no owner is found.
func TestPackageOwner_Executes(t *testing.T) {
	name, ver, err := PackageOwner("/bin/sh")
	if err != nil {
		assert.ErrorIs(t, err, ErrNoPackage)
		assert.Empty(t, name)
		assert.Empty(t, ver)
		return
	}
	assert.NotEmpty(t, name)
}
