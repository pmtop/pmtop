package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pmtop/pmtop/internal/export"
)

// exportView writes the current (filtered) view to a file in the user's
// current directory as JSON or CSV (PRD FR-08-05). fmt controls the format:
// "json" or "csv". Returns the written path and any error.
func (m *Model) exportView(fmtSpec string) (string, error) {
	if fmtSpec != "json" && fmtSpec != "csv" {
		fmtSpec = "json"
	}
	name := fmt.Sprintf("pmtop-export-%s.%s", time.Now().Format("20060102-150405"), fmtSpec)
	path := filepath.Join(".", name)

	var data []byte
	var err error
	if fmtSpec == "csv" {
		data, err = export.CSV(m.socks)
	} else {
		data, err = export.JSON(m.socks)
	}
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// doExport performs an export and updates the status bar with the result.
func (m *Model) doExport() {
	if len(m.socks) == 0 {
		m.setStatus("nothing to export", 2*time.Second)
		return
	}
	path, err := m.exportView("json")
	if err != nil {
		m.setStatus("export failed: "+err.Error(), 3*time.Second)
		return
	}
	m.setStatus("exported "+path, 3*time.Second)
}
