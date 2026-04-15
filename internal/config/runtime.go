package config

import (
	"fmt"
	"path/filepath"

	"github.com/peaberberian/paul-envs/internal/utils"
)

// RuntimeConfig holds the parsed contents of a run.conf file.
type RuntimeConfig struct {
	Version      utils.Version
	ProjectPath  string
	Volumes      []string
	Ports        []string
	WorkDir      string // optional; defaults to the project mount target if empty
	DotfilesPath string // optional; if set, mounted read-only and synced into $HOME on start
	GitName      string // optional; applied to git/jj at container start
	GitEmail     string // optional; applied to git/jj at container start
}

// LoadRuntimeConfig parses the run.conf file at path and returns a
// RuntimeConfig. It returns an error if the file cannot be parsed or if the
// required PATH directive is absent.
func LoadRuntimeConfig(path string) (RuntimeConfig, error) {
	directives, err := ParseFile(path)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("load runtime config %s: %w", filepath.Base(path), err)
	}
	version, directives, err := extractVersion(path, directives, expectedRuntimeConfigVersion())
	if err != nil {
		return RuntimeConfig{}, err
	}

	cfg := RuntimeConfig{Version: version}

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
		case "DOTFILES_PATH":
			cfg.DotfilesPath = d.Value
		case "GIT_AUTHOR_NAME":
			cfg.GitName = d.Value
		case "GIT_AUTHOR_EMAIL":
			cfg.GitEmail = d.Value
		default:
			return RuntimeConfig{}, fmt.Errorf("%s: unknown directive %q", filepath.Base(path), d.Key)
		}
	}

	if cfg.ProjectPath == "" {
		return RuntimeConfig{}, fmt.Errorf("%s: required directive PATH is missing", filepath.Base(path))
	}

	return cfg, nil
}
