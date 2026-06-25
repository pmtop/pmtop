// Package config implements layered configuration loading for pmtop
// (PRD FR-08): system config (/etc/pmtop/config.toml) < user config
// (~/.config/pmtop/config.toml) < CLI flags.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds all configurable pmtop settings.
type Config struct {
	RefreshInterval string  `toml:"refresh_interval"`
	SortColumn      string  `toml:"sort_column"`
	SortAsc         bool    `toml:"sort_asc"`
	ColorblindMode  bool    `toml:"colorblind_mode"`
	NoColor         bool    `toml:"no_color"`
	DockerSocket    string  `toml:"docker_socket"`
	Keybindings     map[string]string `toml:"keybindings"`
}

// Default returns the built-in defaults (PRD FR-08-04).
func Default() Config {
	return Config{
		RefreshInterval: "2s",
		SortColumn:      "proto",
		SortAsc:         true,
		ColorblindMode:  false,
		NoColor:         false,
		DockerSocket:    "/var/run/docker.sock",
		Keybindings:     map[string]string{},
	}
}

// Interval parses RefreshInterval into a Duration, falling back to 2s.
func (c Config) Interval() time.Duration {
	d, err := time.ParseDuration(c.RefreshInterval)
	if err != nil || d <= 0 {
		return 2 * time.Second
	}
	return d
}

// SystemPath returns the system config path.
func SystemPath() string { return "/etc/pmtop/config.toml" }

// UserPath returns the user config path following XDG (FR-08-02).
func UserPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "pmtop", "config.toml")
}

// Load reads system then user config, applying defaults first (FR-08-03).
// Missing files are not errors; each layer overrides the previous.
func Load() (Config, error) {
	cfg := Default()
	var firstErr error
	for _, path := range []string{SystemPath(), UserPath()} {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue // missing file is fine
		}
		if err := mergeFile(&cfg, path); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	// Honor NO_COLOR env (FR-08-06): disables colors even if config sets it.
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		cfg.NoColor = true
	}
	return cfg, firstErr
}

// mergeFile decodes path into a temp Config and merges non-zero values into
// the running config.
func mergeFile(cfg *Config, path string) error {
	var f Config
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return fmt.Errorf("config %s: %w", path, err)
	}
	// Only override when the file actually set the field. TOML decode leaves
	// unset fields at their zero value, so we re-decode into a map to detect.
	var raw map[string]interface{}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return err
	}
	if v, ok := raw["refresh_interval"]; ok {
		if s, ok := v.(string); ok {
			cfg.RefreshInterval = s
		}
	}
	if v, ok := raw["sort_column"]; ok {
		if s, ok := v.(string); ok {
			cfg.SortColumn = s
		}
	}
	if v, ok := raw["sort_asc"]; ok {
		if b, ok := v.(bool); ok {
			cfg.SortAsc = b
		}
	}
	if v, ok := raw["colorblind_mode"]; ok {
		if b, ok := v.(bool); ok {
			cfg.ColorblindMode = b
		}
	}
	if v, ok := raw["no_color"]; ok {
		if b, ok := v.(bool); ok {
			cfg.NoColor = b
		}
	}
	if v, ok := raw["docker_socket"]; ok {
		if s, ok := v.(string); ok {
			cfg.DockerSocket = s
		}
	}
	if kb, ok := raw["keybindings"]; ok {
		if m, ok := kb.(map[string]interface{}); ok {
			if cfg.Keybindings == nil {
				cfg.Keybindings = map[string]string{}
			}
			for k, val := range m {
				cfg.Keybindings[k] = fmt.Sprint(val)
			}
		}
	}
	_ = f // f was used to validate the schema; raw provides exact values
	return nil
}

// ApplyFlags overlays CLI flag values onto the config (FR-08-03). Non-zero /
// non-empty flag values win.
func ApplyFlags(cfg Config, flags Flags) Config {
	if flags.RefreshInterval != "" {
		cfg.RefreshInterval = flags.RefreshInterval
	}
	if flags.SortColumn != "" {
		cfg.SortColumn = flags.SortColumn
	}
	if flags.SortAscProvided {
		cfg.SortAsc = flags.SortAsc
	}
	if flags.ColorblindProvided {
		cfg.ColorblindMode = flags.ColorblindMode
	}
	if flags.NoColor {
		cfg.NoColor = true
	}
	if flags.DockerSocket != "" {
		cfg.DockerSocket = flags.DockerSocket
	}
	return cfg
}

// Flags holds CLI flag values to overlay onto Config. The *Provided fields
// distinguish "flag not set" from "flag set to false".
type Flags struct {
	RefreshInterval   string
	SortColumn        string
	SortAsc           bool
	SortAscProvided   bool
	ColorblindMode    bool
	ColorblindProvided bool
	NoColor           bool
	DockerSocket      string
}

// ErrInvalidInterval is returned by Validate when the interval can't parse.
var ErrInvalidInterval = errors.New("invalid refresh_interval")

// Validate checks the config for obvious problems.
func (c Config) Validate() error {
	switch strings.ToLower(c.RefreshInterval) {
	case "manual":
		return nil
	}
	if _, err := time.ParseDuration(c.RefreshInterval); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInterval, c.RefreshInterval)
	}
	return nil
}

// Summary returns a one-line description of the active config (for debug).
func (c Config) Summary() string {
	return fmt.Sprintf("interval=%s sort=%s/%v colorblind=%v nocolor=%v docker=%s",
		c.RefreshInterval, c.SortColumn, c.SortAsc, c.ColorblindMode, c.NoColor, c.DockerSocket)
}

// ParseIntervalFlag converts a user interval token (e.g. "2s", "500ms",
// "manual") to a Duration. "manual" returns 0 (no auto-refresh).
func ParseIntervalFlag(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 2 * time.Second, nil
	}
	if strings.EqualFold(s, "manual") {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid interval %q: %w", s, err)
	}
	return d, nil
}

// ParseSortColumn maps a column name to a normalized token.
func ParseSortColumn(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "proto", "protocol":
		return "proto", nil
	case "port", "local", "remote", "state", "pid", "process", "container":
		return s, nil
	}
	return "", fmt.Errorf("unknown sort column %q", s)
}

// ParseBoolFlag parses common boolean spellings from flags/config.
func ParseBoolFlag(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off", "":
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean %q", s)
}

// AtoiSafe is a small helper for parsing optional numeric config values.
func AtoiSafe(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}
