package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Cleanup(func() { os.Remove(path) })
}

func TestDefault(t *testing.T) {
	c := Default()
	assert.Equal(t, "2s", c.RefreshInterval)
	assert.Equal(t, "proto", c.SortColumn)
	assert.True(t, c.SortAsc)
	assert.False(t, c.ColorblindMode)
	assert.Equal(t, "/var/run/docker.sock", c.DockerSocket)
}

func TestInterval(t *testing.T) {
	assert.Equal(t, 2*time.Second, Config{RefreshInterval: "2s"}.Interval())
	assert.Equal(t, 2*time.Second, Config{RefreshInterval: "bad"}.Interval(), "fallback to 2s")
	assert.Equal(t, 500*time.Millisecond, Config{RefreshInterval: "500ms"}.Interval())
	assert.Equal(t, 5*time.Second, Config{RefreshInterval: "5s"}.Interval())
}

func TestLoad_UserOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	usr := filepath.Join(dir, "user.toml")
	writeConfig(t, usr, `refresh_interval = "5s"
sort_column = "port"
sort_asc = false
colorblind_mode = true
`)
	// Point UserPath at the temp file via a helper: replace the function for
	// this test by setting XDG_CONFIG_HOME so UserPath() resolves under dir.
	// We instead call mergeFile directly to avoid global mutation.
	cfg := Default()
	require.NoError(t, mergeFile(&cfg, usr))
	assert.Equal(t, "5s", cfg.RefreshInterval)
	assert.Equal(t, "port", cfg.SortColumn)
	assert.False(t, cfg.SortAsc)
	assert.True(t, cfg.ColorblindMode)
}

func TestLoad_SystemThenUser(t *testing.T) {
	dir := t.TempDir()
	sysF := filepath.Join(dir, "sys.toml")
	usrF := filepath.Join(dir, "user.toml")
	writeConfig(t, sysF, `refresh_interval = "1s"
sort_column = "state"
`)
	writeConfig(t, usrF, `refresh_interval = "5s"
`)
	cfg := Default()
	require.NoError(t, mergeFile(&cfg, sysF))
	require.NoError(t, mergeFile(&cfg, usrF))
	// User overrides system for refresh_interval; system's sort_column stays.
	assert.Equal(t, "5s", cfg.RefreshInterval)
	assert.Equal(t, "state", cfg.SortColumn)
}

func TestLoad_HonorsNoColorEnv(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	t.Cleanup(func() { os.Unsetenv("NO_COLOR") })
	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.NoColor, "NO_COLOR env forces no_color")
}

func TestApplyFlags(t *testing.T) {
	cfg := Default()
	flags := Flags{
		RefreshInterval:   "1s",
		SortColumn:        "pid",
		SortAsc:           false,
		SortAscProvided:   true,
		ColorblindMode:    true,
		ColorblindProvided: true,
		NoColor:           true,
		DockerSocket:      "/tmp/docker.sock",
	}
	cfg = ApplyFlags(cfg, flags)
	assert.Equal(t, "1s", cfg.RefreshInterval)
	assert.Equal(t, "pid", cfg.SortColumn)
	assert.False(t, cfg.SortAsc)
	assert.True(t, cfg.ColorblindMode)
	assert.True(t, cfg.NoColor)
	assert.Equal(t, "/tmp/docker.sock", cfg.DockerSocket)
}

func TestValidate(t *testing.T) {
	assert.NoError(t, Config{RefreshInterval: "2s"}.Validate())
	assert.NoError(t, Config{RefreshInterval: "manual"}.Validate())
	assert.Error(t, Config{RefreshInterval: "nope"}.Validate())
}

func TestParseIntervalFlag(t *testing.T) {
	d, err := ParseIntervalFlag("1s")
	require.NoError(t, err)
	assert.Equal(t, time.Second, d)

	d, err = ParseIntervalFlag("manual")
	require.NoError(t, err)
	assert.Zero(t, d, "manual -> 0 (no auto refresh)")

	d, err = ParseIntervalFlag("")
	require.NoError(t, err)
	assert.Equal(t, 2*time.Second, d, "empty -> default 2s")

	_, err = ParseIntervalFlag("zzz")
	assert.Error(t, err)
}

func TestParseSortColumn(t *testing.T) {
	c, err := ParseSortColumn("PID")
	require.NoError(t, err)
	assert.Equal(t, "pid", c)

	c, err = ParseSortColumn("protocol")
	require.NoError(t, err)
	assert.Equal(t, "proto", c)

	_, err = ParseSortColumn("bogus")
	assert.Error(t, err)
}

func TestParseBoolFlag(t *testing.T) {
	for _, in := range []string{"true", "yes", "1", "on", "TRUE"} {
		v, err := ParseBoolFlag(in)
		assert.NoError(t, err, in)
		assert.True(t, v, in)
	}
	for _, in := range []string{"false", "no", "0", "off", ""} {
		v, err := ParseBoolFlag(in)
		assert.NoError(t, err, in)
		assert.False(t, v, in)
	}
	_, err := ParseBoolFlag("maybe")
	assert.Error(t, err)
}

func TestSummary(t *testing.T) {
	c := Default()
	s := c.Summary()
	assert.Contains(t, s, "interval=2s")
	assert.Contains(t, s, "sort=proto")
}
