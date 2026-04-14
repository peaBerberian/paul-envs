// Package config parses build.conf and run.conf files.
// Both formats share the same syntax:
//   - Blank lines are ignored.
//   - Lines whose first non-space character is '#' are ignored (comments).
//   - Every significant line has the form: DIRECTIVE VALUE
//     where DIRECTIVE and VALUE are separated by the first space character.
//     Everything after that first space is the value (may contain spaces).
//   - A leading '~' in a value is expanded to the current user's home directory.
//   - Order is preserved and duplicate directives are allowed.
package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Directive is a single key/value pair parsed from a conf file.
type Directive struct {
	Key   string
	Value string
}

// Parse reads all directives from r.
// It expands a leading '~' in each value to the user's home directory.
// Lines that are blank or start with '#' (after trimming leading spaces) are
// silently skipped. A line that contains no space character is an error.
func Parse(r io.Reader) ([]Directive, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		// Non-fatal: we simply won't be able to expand '~'.
		home = ""
	}
	return parse(r, home)
}

// ParseFile is a convenience wrapper around Parse that opens the named file.
func ParseFile(path string) ([]Directive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// parse is the internal implementation, accepting an explicit homeDir so tests
// can inject a predictable value without touching the real filesystem.
func parse(r io.Reader, homeDir string) ([]Directive, error) {
	var directives []Directive
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip blank lines and comments.
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}

		// Split on the first space only.
		key, value, ok := strings.Cut(trimmed, " ")
		if !ok {
			return nil, fmt.Errorf("config: line %d: missing value (no space found): %q", lineNum, trimmed)
		}

		// Expand a leading '~'.
		if homeDir != "" && strings.HasPrefix(value, "~/") {
			value = homeDir + value[1:]
		} else if value == "~" {
			value = homeDir
		}

		directives = append(directives, Directive{Key: key, Value: value})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("config: scanner error: %w", err)
	}

	return directives, nil
}
