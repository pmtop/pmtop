package process

import (
	"os/exec"
	"strings"
)

// PackageOwner identifies the system package that owns the given executable
// path (FR-04-05). It tries dpkg -S first (Debian/Ubuntu) and falls back to
// rpm -qf (RHEL/Fedora/Rocky/Alma). Returns the package name and version
// (version is best-effort and may be empty).
func PackageOwner(path string) (name, version string, err error) {
	if name, ok := dpkgOwner(path); ok {
		return name, "", nil
	}
	if name, ver, ok := rpmOwner(path); ok {
		return name, ver, nil
	}
	return "", "", ErrNoPackage
}

// ErrNoPackage is returned when no owning package can be determined.
var ErrNoPackage = packageErr("no owning package found")

type packageErr string

func (e packageErr) Error() string { return string(e) }

// dpkgOwner runs `dpkg -S path` and parses the output.
func dpkgOwner(path string) (string, bool) {
	out, err := exec.Command("dpkg", "-S", path).Output()
	if err != nil {
		return "", false
	}
	return ParseDpkgS(string(out))
}

// rpmOwner runs `rpm -qf --qf '%{NAME} %{VERSION}-%{RELEASE}' path`.
func rpmOwner(path string) (string, string, bool) {
	out, err := exec.Command("rpm", "-qf", "--qf", "%{NAME} %{VERSION}-%{RELEASE}", path).Output()
	if err != nil {
		return "", "", false
	}
	return ParseRpmQf(string(out))
}

// ParseDpkgS parses "pkgname: /path/to/file" output. dpkg -S may return
// multiple owners comma-separated; the first is returned.
func ParseDpkgS(out string) (string, bool) {
	out = strings.TrimSpace(out)
	if out == "" || strings.Contains(out, "no path found") {
		return "", false
	}
	line := strings.SplitN(out, "\n", 2)[0]
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return "", false
	}
	owners := strings.Split(parts[0], ",")
	return strings.TrimSpace(owners[0]), owners[0] != ""
}

// ParseRpmQf parses "name version-release" output from rpm -qf.
func ParseRpmQf(out string) (string, string, bool) {
	out = strings.TrimSpace(out)
	if out == "" || strings.Contains(out, "not installed") || strings.HasPrefix(out, "file ") {
		return "", "", false
	}
	fields := strings.Fields(out)
	if len(fields) >= 2 {
		return fields[0], fields[1], true
	}
	if len(fields) == 1 {
		return fields[0], "", true
	}
	return "", "", false
}
