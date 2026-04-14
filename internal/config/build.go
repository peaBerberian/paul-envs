package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildConfig holds the validated build-time arguments to pass to the
// Dockerfile. Args maps directive names (e.g. "HOST_UID") to their values
// exactly as they appear in build.conf. The engine layer forwards each entry
// as a --build-arg KEY=VALUE flag without further interpretation.
type BuildConfig struct {
	Args map[string]string
}

// requiredBuildDirectives must be present in every build.conf.
var requiredBuildDirectives = map[string]struct{}{
	"HOST_UID":   {},
	"HOST_GID":   {},
	"USERNAME":   {},
	"USER_SHELL": {},
}

// boolBuildDirectives must have the value "true" or "false".
var boolBuildDirectives = map[string]struct{}{
	"INSTALL_NEOVIM":      {},
	"INSTALL_STARSHIP":    {},
	"INSTALL_OH_MY_POSH":  {},
	"INSTALL_ATUIN":       {},
	"INSTALL_MISE":        {},
	"INSTALL_ZELLIJ":      {},
	"INSTALL_JUJUTSU":     {},
	"INSTALL_DELTA":       {},
	"INSTALL_OPEN_CODE":   {},
	"INSTALL_CLAUDE_CODE": {},
	"INSTALL_CODEX":       {},
	"INSTALL_FIREFOX":     {},
	"ENABLE_WASM":         {},
	"ENABLE_SSH":          {},
	"ENABLE_SUDO":         {},
}

// versionBuildDirectives must be "none" or a non-empty, whitespace-free string.
var versionBuildDirectives = map[string]struct{}{
	"INSTALL_NODE":   {},
	"INSTALL_RUST":   {},
	"INSTALL_PYTHON": {},
	"INSTALL_GO":     {},
}

// allBuildDirectives is the union of all recognised directive names.
var allBuildDirectives = func() map[string]struct{} {
	m := map[string]struct{}{
		"GIT_AUTHOR_NAME":        {},
		"GIT_AUTHOR_EMAIL":       {},
		"SUPPLEMENTARY_PACKAGES": {},
		"DOTFILES_DIR":           {},
	}
	for k := range requiredBuildDirectives {
		m[k] = struct{}{}
	}
	for k := range boolBuildDirectives {
		m[k] = struct{}{}
	}
	for k := range versionBuildDirectives {
		m[k] = struct{}{}
	}
	return m
}()

// LoadBuildConfig parses path as a build.conf file, validates its contents,
// and returns a BuildConfig whose Args are ready to forward as --build-arg
// flags. Any validation failure returns a descriptive error.
func LoadBuildConfig(path string) (BuildConfig, error) {
	directives, err := ParseFile(path)
	if err != nil {
		return BuildConfig{}, fmt.Errorf("load build config %s: %w", filepath.Base(path), err)
	}

	args := make(map[string]string, len(directives))

	for _, d := range directives {
		if _, known := allBuildDirectives[d.Key]; !known {
			fmt.Fprintf(os.Stderr, "Warning: %s: ignoring unknown directive %q\n", filepath.Base(path), d.Key)
			continue
		}

		if _, duplicate := args[d.Key]; duplicate {
			return BuildConfig{}, fmt.Errorf(
				"%s: directive %q appears more than once",
				filepath.Base(path), d.Key,
			)
		}

		if err := validateBuildValue(d.Key, d.Value); err != nil {
			return BuildConfig{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}

		args[d.Key] = d.Value
	}

	for k := range requiredBuildDirectives {
		if _, present := args[k]; !present {
			return BuildConfig{}, fmt.Errorf(
				"%s: missing required directive %q\n"+
					"hint: re-run 'paul-env create' to regenerate your build.conf",
				filepath.Base(path), k,
			)
		}
	}

	return BuildConfig{Args: args}, nil
}

// validateBuildValue checks that value is acceptable for the given directive.
func validateBuildValue(key, value string) error {
	if _, ok := boolBuildDirectives[key]; ok {
		if value != "true" && value != "false" {
			return fmt.Errorf("directive %q: expected \"true\" or \"false\", got %q", key, value)
		}
		return nil
	}

	if _, ok := versionBuildDirectives[key]; ok {
		if value == "none" {
			return nil
		}
		if value == "" || strings.ContainsAny(value, " \t") {
			return fmt.Errorf("directive %q: expected \"none\" or a non-empty version string without whitespace, got %q", key, value)
		}
		return nil
	}

	return nil
}
