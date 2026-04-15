package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempConf writes content to a temporary build.conf and returns its path.
func writeTempConf(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "build.conf")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	return f.Name()
}

// minimalValidConf returns the minimum valid build.conf content: all required
// directives set, all bool/version directives at their default off values.
const minimalValidConf = `HOST_UID 1000
VERSION 1.0.0
HOST_GID 1000
USERNAME dev
USER_SHELL bash
INSTALL_NEOVIM false
INSTALL_STARSHIP false
INSTALL_OH_MY_POSH false
INSTALL_ATUIN false
INSTALL_MISE false
INSTALL_ZELLIJ false
INSTALL_JUJUTSU false
INSTALL_DELTA false
INSTALL_OPEN_CODE false
INSTALL_CLAUDE_CODE false
INSTALL_CODEX false
INSTALL_FIREFOX false
ENABLE_WASM false
ENABLE_SSH false
ENABLE_SUDO false
INSTALL_NODE none
INSTALL_RUST none
INSTALL_PYTHON none
INSTALL_GO none
`

func TestLoadBuildConfig_Valid(t *testing.T) {
	path := writeTempConf(t, minimalValidConf)
	cfg, err := LoadBuildConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct{ key, want string }{
		{"HOST_UID", "1000"},
		{"HOST_GID", "1000"},
		{"USERNAME", "dev"},
		{"USER_SHELL", "bash"},
		{"INSTALL_NEOVIM", "false"},
		{"INSTALL_NODE", "none"},
	}
	for _, c := range cases {
		if got := cfg.Args[c.key]; got != c.want {
			t.Errorf("Args[%q] = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestLoadBuildConfig_CommentsAndBlanksIgnored(t *testing.T) {
	content := `# This is a comment

HOST_UID 1000
VERSION 1.0.0
# Another comment
HOST_GID 1000
USERNAME dev
USER_SHELL bash

INSTALL_NEOVIM false
INSTALL_STARSHIP false
INSTALL_OH_MY_POSH false
INSTALL_ATUIN false
INSTALL_MISE false
INSTALL_ZELLIJ false
INSTALL_JUJUTSU false
INSTALL_DELTA false
INSTALL_OPEN_CODE false
INSTALL_CLAUDE_CODE false
INSTALL_CODEX false
INSTALL_FIREFOX false
ENABLE_WASM false
ENABLE_SSH false
ENABLE_SUDO false
INSTALL_NODE none
INSTALL_RUST none
INSTALL_PYTHON none
INSTALL_GO none
`
	path := writeTempConf(t, content)
	_, err := LoadBuildConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadBuildConfig_FreeformDirectives(t *testing.T) {
	content := minimalValidConf +
		"SUPPLEMENTARY_PACKAGES curl wget\n"

	path := writeTempConf(t, content)
	cfg, err := LoadBuildConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct{ key, want string }{
		{"SUPPLEMENTARY_PACKAGES", "curl wget"},
	}
	for _, c := range cases {
		if got := cfg.Args[c.key]; got != c.want {
			t.Errorf("Args[%q] = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestLoadBuildConfig_BoolDirectives(t *testing.T) {
	// true is also valid
	content := minimalValidConf
	// override INSTALL_NEOVIM to true by replacing the line
	// (writeTempConf just needs a valid file, so build a fresh one)
	content2 := `HOST_UID 1000
VERSION 1.0.0
HOST_GID 1000
USERNAME dev
USER_SHELL bash
INSTALL_NEOVIM true
INSTALL_STARSHIP false
INSTALL_OH_MY_POSH false
INSTALL_ATUIN false
INSTALL_MISE false
INSTALL_ZELLIJ false
INSTALL_JUJUTSU false
INSTALL_DELTA false
INSTALL_OPEN_CODE false
INSTALL_CLAUDE_CODE false
INSTALL_CODEX false
INSTALL_FIREFOX false
ENABLE_WASM false
ENABLE_SSH true
ENABLE_SUDO false
INSTALL_NODE none
INSTALL_RUST none
INSTALL_PYTHON none
INSTALL_GO none
`
	_ = content
	path := writeTempConf(t, content2)
	cfg, err := LoadBuildConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Args["INSTALL_NEOVIM"] != "true" {
		t.Errorf("INSTALL_NEOVIM = %q, want \"true\"", cfg.Args["INSTALL_NEOVIM"])
	}
	if cfg.Args["ENABLE_SSH"] != "true" {
		t.Errorf("ENABLE_SSH = %q, want \"true\"", cfg.Args["ENABLE_SSH"])
	}
}

func TestLoadBuildConfig_VersionDirectives(t *testing.T) {
	cases := []struct {
		directive string
		value     string
	}{
		{"INSTALL_NODE", "lts"},
		{"INSTALL_NODE", "lts/hydrogen"},
		{"INSTALL_NODE", "20"},
		{"INSTALL_NODE", "20.11.0"},
		{"INSTALL_RUST", "stable"},
		{"INSTALL_PYTHON", "3.12.0"},
		{"INSTALL_GO", "1.23.0"},
	}

	for _, c := range cases {
		base := buildConfWithOverride(t, c.directive, c.value)
		path := writeTempConf(t, base)
		cfg, err := LoadBuildConfig(path)
		if err != nil {
			t.Errorf("%s=%q: unexpected error: %v", c.directive, c.value, err)
			continue
		}
		if got := cfg.Args[c.directive]; got != c.value {
			t.Errorf("%s=%q: got %q", c.directive, c.value, got)
		}
	}
}

// buildConfWithOverride returns a minimal valid conf with one version directive
// overridden to value.
// buildConfWithOverride returns a minimal valid conf with one directive overridden.
func buildConfWithOverride(t *testing.T, directive, value string) string {
	t.Helper()

	base := minimalConf()

	lines := strings.Split(base, "\n")
	found := false

	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == directive {
			lines[i] = directive + " " + value
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("buildConfWithOverride: directive %q not found in base conf", directive)
	}

	return strings.Join(lines, "\n")
}

// minimalConf returns the baseline configuration used in tests.
func minimalConf() string {
	return `HOST_UID 1000
VERSION 1.0.0
HOST_GID 1000
USERNAME dev
USER_SHELL bash
INSTALL_NEOVIM false
INSTALL_STARSHIP false
INSTALL_OH_MY_POSH false
INSTALL_ATUIN false
INSTALL_MISE false
INSTALL_ZELLIJ false
INSTALL_JUJUTSU false
INSTALL_DELTA false
INSTALL_OPEN_CODE false
INSTALL_CLAUDE_CODE false
INSTALL_CODEX false
INSTALL_FIREFOX false
ENABLE_WASM false
ENABLE_SSH false
ENABLE_SUDO false
INSTALL_NODE none
INSTALL_RUST none
INSTALL_PYTHON none
INSTALL_GO none
`
}

func replaceFirst(s, old, new string) string {
	idx := 0
	for i := range s {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
		_ = idx
	}
	return s
}

// --- Error cases ---

func TestLoadBuildConfig_UnknownDirectiveIgnored(t *testing.T) {
	content := minimalValidConf + "UNKNOWN_THING foo\n"
	path := writeTempConf(t, content)
	cfg, err := LoadBuildConfig(path)
	if err != nil {
		t.Fatalf("unexpected error for unknown directive: %v", err)
	}
	if _, ok := cfg.Args["UNKNOWN_THING"]; ok {
		t.Fatal("unknown directive should not be forwarded as a build arg")
	}
}

func TestLoadBuildConfig_MissingVersion(t *testing.T) {
	content := confWithoutDirective(t, minimalValidConf, "VERSION")
	path := writeTempConf(t, content)
	_, err := LoadBuildConfig(path)
	if err == nil {
		t.Fatal("expected error for missing VERSION, got nil")
	}
	assertErrorContains(t, err, "missing required directive VERSION")
}

func TestLoadBuildConfig_DuplicateDirective(t *testing.T) {
	content := minimalValidConf + "HOST_UID 2000\n"
	path := writeTempConf(t, content)
	_, err := LoadBuildConfig(path)
	if err == nil {
		t.Fatal("expected error for duplicate directive, got nil")
	}
	assertErrorContains(t, err, "more than once")
	assertErrorContains(t, err, "HOST_UID")
}

func TestLoadBuildConfig_MissingRequired(t *testing.T) {
	required := []string{"HOST_UID", "HOST_GID", "USERNAME", "USER_SHELL"}
	for _, directive := range required {
		t.Run(directive, func(t *testing.T) {
			content := confWithoutDirective(t, minimalValidConf, directive)
			path := writeTempConf(t, content)
			_, err := LoadBuildConfig(path)
			if err == nil {
				t.Fatalf("expected error for missing %q, got nil", directive)
			}
			assertErrorContains(t, err, "missing required directive")
			assertErrorContains(t, err, directive)
			assertErrorContains(t, err, "paul-env create")
		})
	}
}

func TestLoadBuildConfig_BoolBadValue(t *testing.T) {
	boolDirectives := []string{
		"INSTALL_NEOVIM", "INSTALL_STARSHIP", "ENABLE_WASM", "ENABLE_SSH", "ENABLE_SUDO",
	}
	badValues := []string{"yes", "no", "1", "0", "TRUE", "FALSE", ""}
	for _, directive := range boolDirectives {
		for _, bad := range badValues {
			t.Run(directive+"="+bad, func(t *testing.T) {
				content := buildConfWithOverride(t, directive, bad)
				// buildConfWithOverride only replaces "none" lines; bool
				// directives default to "false", so build the override manually.
				content = buildConfWithBoolOverride(t, directive, bad)
				path := writeTempConf(t, content)
				_, err := LoadBuildConfig(path)
				if err == nil {
					t.Fatalf("expected error for %s=%q, got nil", directive, bad)
				}
				assertErrorContains(t, err, directive)
				assertErrorContains(t, err, "true\" or \"false")
			})
		}
	}
}

// buildConfWithBoolOverride replaces a bool directive's "false" value.
func buildConfWithBoolOverride(t *testing.T, directive, value string) string {
	t.Helper()
	old := directive + " false"
	new := directive + " " + value
	result := replaceFirst(minimalValidConf, old, new)
	if result == minimalValidConf {
		t.Fatalf("buildConfWithBoolOverride: %q not found", directive)
	}
	return result
}

func TestLoadBuildConfig_VersionBadValue(t *testing.T) {
	cases := []struct {
		directive string
		value     string
	}{
		{"INSTALL_NODE", "lts hydrogen"}, // space
		{"INSTALL_NODE", "20\t11"},       // tab
		{"INSTALL_RUST", ""},             // empty (requires non-empty if not none)
	}
	for _, c := range cases {
		t.Run(c.directive+"="+c.value, func(t *testing.T) {
			content := buildConfWithOverride(t, c.directive, c.value)
			path := writeTempConf(t, content)
			_, err := LoadBuildConfig(path)
			if err == nil {
				t.Fatalf("expected error for %s=%q, got nil", c.directive, c.value)
			}
			assertErrorContains(t, err, c.directive)
		})
	}
}

func TestLoadBuildConfig_EmptyFile(t *testing.T) {
	path := writeTempConf(t, "")
	_, err := LoadBuildConfig(path)
	if err == nil {
		t.Fatal("expected error for empty file (missing required directives), got nil")
	}
	assertErrorContains(t, err, "missing required directive")
}

func TestLoadBuildConfig_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.conf")
	_, err := LoadBuildConfig(path)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// --- helpers ---

func assertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected non-nil error containing %q", substr)
	}
	if !contains(err.Error(), substr) {
		t.Errorf("error %q does not contain %q", err.Error(), substr)
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// confWithoutDirective removes all lines starting with directive from content.
func confWithoutDirective(t *testing.T, content, directive string) string {
	t.Helper()
	var out []byte
	lines := splitLines(content)
	prefix := directive + " "
	for _, line := range lines {
		if len(line) >= len(prefix) && line[:len(prefix)] == prefix {
			continue
		}
		if line == directive {
			continue
		}
		out = append(out, []byte(line+"\n")...)
	}
	return string(out)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
