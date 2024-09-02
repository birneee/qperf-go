package qlog

import (
	"fmt"
	"os"
	"path/filepath"
)

// NewQlogDirWriter returns nil if QLOGDIR environment variable is not set.
// id should be a byte sequence that is unique enough to not cause any name collisions, e.g. the QUIC ODCID.
// label is a descriptive name or category e.g. client.
func NewQlogDirWriter(id []byte, label string, config *Config) Writer {
	qlogDir := os.Getenv("QLOGDIR")
	if qlogDir == "" {
		return nil
	}
	path := filepath.Join(
		qlogDir,
		fmt.Sprintf("%x_%s.qlog", id, label))
	return NewFileQlogWriter(path, config)
}
