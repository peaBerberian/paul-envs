package engine

import "strings"

func containerTerminalEnvArgs(termName, colorTerm string) []string {
	termName = strings.ToLower(strings.TrimSpace(termName))
	colorTerm = strings.ToLower(strings.TrimSpace(colorTerm))

	if termName == "dumb" {
		return []string{"--env", "TERM=dumb"}
	}

	// Host-specific terminal names are often missing from minimal images.
	// Use the broadly available xterm entries while preserving color depth.
	containerTerm := "xterm"
	if colorTerm == "truecolor" || colorTerm == "24bit" ||
		strings.Contains(termName, "256color") || strings.Contains(termName, "direct") {
		containerTerm = "xterm-256color"
	}

	args := []string{"--env", "TERM=" + containerTerm}
	if colorTerm == "truecolor" || colorTerm == "24bit" {
		args = append(args, "--env", "COLORTERM="+colorTerm)
	}
	return args
}
