//go:build linux

package elevate

import "os"

const isLinux = true

func init() {
	currentEUID = os.Geteuid
}
