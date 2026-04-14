package config

import (
	"fmt"
	"path/filepath"
)

// RuntimeConfig holds the parsed contents of a run.conf file.
type RuntimeConfig struct {
	ProjectPath string
	Volumes     []string
	Ports       []string
	WorkDir     string // optional; defaults to the project mount target if empty
}

// LoadRuntimeConfig parses the run.conf file at path and returns a
// RuntimeConfig. It returns an error if the file cannot be parsed or if the
// required PATH directive is absent.
func LoadRuntimeConfig(path string) (RuntimeConfig, error) {
	directives, err := ParseFile(path)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("load runtime config %s: %w", filepath.Base(path), err)
	}

	var cfg RuntimeConfig

	for _, d := range directives {
		switch d.Key {
		case "PATH":
			cfg.ProjectPath = d.Value
		case "VOLUME":
			cfg.Volumes = append(cfg.Volumes, d.Value)
		case "PORT":
			cfg.Ports = append(cfg.Ports, d.Value)
		case "WORKDIR":
			cfg.WorkDir = d.Value
		default:
			continue
		}
	}

	if cfg.ProjectPath == "" {
		return RuntimeConfig{}, fmt.Errorf("%s: required directive PATH is missing", filepath.Base(path))
	}

	return cfg, nil
}
