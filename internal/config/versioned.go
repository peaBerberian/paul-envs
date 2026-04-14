package config

import (
	"fmt"
	"path/filepath"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func extractVersion(path string, directives []Directive, expected utils.Version) (utils.Version, []Directive, error) {
	var (
		found   *utils.Version
		rest    []Directive
		seenCnt int
	)

	for _, d := range directives {
		if d.Key != "VERSION" {
			rest = append(rest, d)
			continue
		}

		seenCnt++
		parsed, err := utils.ParseVersion(d.Value)
		if err != nil {
			return utils.Version{}, nil, fmt.Errorf("%s: invalid VERSION %q: %w", filepath.Base(path), d.Value, err)
		}
		if !parsed.IsCompatibleWithBase(expected) {
			return utils.Version{}, nil, fmt.Errorf(
				"%s: incompatible VERSION %s (expected compatibility with %s)",
				filepath.Base(path), parsed.ToString(), expected.ToString(),
			)
		}
		found = &parsed
	}

	if seenCnt == 0 {
		return utils.Version{}, nil, fmt.Errorf("%s: missing required directive VERSION", filepath.Base(path))
	}
	if seenCnt > 1 {
		return utils.Version{}, nil, fmt.Errorf("%s: directive %q appears more than once", filepath.Base(path), "VERSION")
	}

	return *found, rest, nil
}

func expectedBuildConfigVersion() utils.Version {
	return versions.BuildConfigVersion
}

func expectedRuntimeConfigVersion() utils.Version {
	return versions.RuntimeConfigVersion
}
