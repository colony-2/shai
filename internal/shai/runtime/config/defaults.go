package config

import (
	"errors"
	"fmt"
	"os"

	_ "embed"
)

//go:embed shai.default.yaml
var embeddedDefault []byte

// GetDefaultConfigBytes returns the embedded default configuration as bytes
func GetDefaultConfigBytes() []byte {
	return embeddedDefault
}

// LoadOrDefault loads the config at path, or falls back to the embedded default when missing.
// Returns true when the embedded default was used.
func LoadOrDefault(path string, env map[string]string, vars map[string]string) (*Config, bool, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg, err := loadFromData(embeddedDefault, path, env, vars)
			return cfg, true, err
		}
		return nil, false, fmt.Errorf("stat shai config: %w", err)
	}
	cfg, err := Load(path, env, vars)
	return cfg, false, err
}
