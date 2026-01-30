package defaultconfig

import (
	_ "embed"
)

// Config contains the embedded default .plumber.yaml configuration.
// The file is copied here by the build process (make build).
// Source of truth: .plumber.yaml in the repository root.
//
//go:embed default.yaml
var Config []byte

// Get returns the embedded default configuration as a byte slice.
func Get() []byte {
	return Config
}

// GetString returns the embedded default configuration as a string.
func GetString() string {
	return string(Config)
}
