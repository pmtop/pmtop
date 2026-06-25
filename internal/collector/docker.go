package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ContainerInfo is the resolved metadata for a running container.
type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
	Ports  []PortMapping
}

// PortMapping describes a published container port.
type PortMapping struct {
	PrivatePort uint16
	PublicPort  uint16
	IP          string
	Type        string
}

// ContainerResolver maps a container ID (long or short) to its metadata.
// Implementations may wrap the Docker / Podman / containerd engine API.
type ContainerResolver interface {
	// Resolve returns container metadata for id (full or short form), and
	// whether a matching container was found.
	Resolve(id string) (ContainerInfo, bool)
	// All returns every known container keyed by full ID.
	All() map[string]ContainerInfo
}

// noResolver matches nothing; the zero-value Collector behavior.
type noResolver struct{}

func (noResolver) Resolve(string) (ContainerInfo, bool) { return ContainerInfo{}, false }
func (noResolver) All() map[string]ContainerInfo        { return nil }

// DockerResolver queries the Docker Engine API over a Unix socket (pure Go,
// no CGO, no Docker SDK dependency — PRD risk mitigation).
type DockerResolver struct {
	sockPath string
	client   *http.Client
	once     sync.Once
	cache    map[string]ContainerInfo
	byShort  map[string]string // short id -> full id
	err      error
	available bool
}

// NewDockerResolver returns a resolver backed by the Docker daemon socket.
// sockPath is typically "/var/run/docker.sock".
func NewDockerResolver(sockPath string) *DockerResolver {
	d := &DockerResolver{sockPath: sockPath}
	d.client = &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sockPath)
			},
		},
	}
	return d
}

// ensureLoaded fetches the container list once (lazy) so the first Resolve
// pays the socket round-trip and subsequent calls are cheap.
func (d *DockerResolver) ensureLoaded() {
	d.once.Do(func() {
		body, err := d.get("/containers/json?all=true")
		if err != nil {
			d.err = err
			d.available = false
			return
		}
		d.cache = ParseContainersJSON(body)
		d.byShort = make(map[string]string, len(d.cache))
		for id := range d.cache {
			d.byShort[id[:min(12, len(id))]] = id
		}
		d.available = true
	})
}

// get performs a GET against the Docker Engine API and returns the body.
func (d *DockerResolver) get(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, "http://docker"+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker api %s: %s", path, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Resolve returns metadata for a container id (full or short form).
func (d *DockerResolver) Resolve(id string) (ContainerInfo, bool) {
	d.ensureLoaded()
	if !d.available {
		return ContainerInfo{}, false
	}
	if c, ok := d.cache[id]; ok {
		return c, true
	}
	if full, ok := d.byShort[id]; ok {
		return d.cache[full], true
	}
	// Fall back to prefix match for arbitrary short lengths.
	for full, c := range d.cache {
		if strings.HasPrefix(full, id) {
			return c, true
		}
	}
	return ContainerInfo{}, false
}

// All returns every known container keyed by full ID.
func (d *DockerResolver) All() map[string]ContainerInfo {
	d.ensureLoaded()
	return d.cache
}

// Available reports whether the Docker socket was reachable at load time.
func (d *DockerResolver) Available() bool {
	d.ensureLoaded()
	return d.available
}

// Err returns any error encountered while loading the container list.
func (d *DockerResolver) Err() error {
	if !d.available {
		return d.err
	}
	return nil
}

// ParseContainersJSON decodes the Docker Engine /containers/json response into
// a map keyed by full container ID. Pure function for unit testing.
func ParseContainersJSON(data []byte) map[string]ContainerInfo {
	var raw []struct {
		ID     string   `json:"Id"`
		Names  []string `json:"Names"`
		Image  string   `json:"Image"`
		State  string   `json:"State"`
		Status string   `json:"Status"`
		Ports  []struct {
			IP          string `json:"IP"`
			PrivatePort uint16 `json:"PrivatePort"`
			PublicPort  uint16 `json:"PublicPort"`
			Type        string `json:"Type"`
		} `json:"Ports"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := make(map[string]ContainerInfo, len(raw))
	for _, c := range raw {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ci := ContainerInfo{
			ID:     c.ID,
			Name:   name,
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
		}
		for _, p := range c.Ports {
			ci.Ports = append(ci.Ports, PortMapping{
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				IP:          p.IP,
				Type:        p.Type,
			})
		}
		out[c.ID] = ci
	}
	return out
}
