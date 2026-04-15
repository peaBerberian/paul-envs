package clihelp

import (
	"flag"
	"strings"

	"github.com/peaberberian/paul-envs/internal/console"
)

func PrintDefaults(console *console.Console, flagset *flag.FlagSet) {
	first := true
	flagset.VisitAll(func(f *flag.Flag) {
		if !first {
			console.WriteLn("")
		}
		first = false

		name, usage := flag.UnquoteUsage(f)
		prefix := "-"
		if len(f.Name) > 1 {
			prefix = "--"
		}

		var line strings.Builder
		line.WriteString("  ")
		line.WriteString(prefix)
		line.WriteString(f.Name)
		if name != "" {
			line.WriteString(" ")
			line.WriteString(name)
		}

		console.WriteLn(line.String())
		for _, paragraph := range strings.Split(usage, "\n") {
			paragraph = strings.TrimSpace(paragraph)
			if paragraph == "" {
				console.WriteLn("")
				continue
			}
			for _, wrappedLine := range wrapText(paragraph, 76) {
				console.WriteLn("    %s", wrappedLine)
			}
		}
	})
}

func wrapText(text string, width int) []string {
	if text == "" || width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	lines := []string{}
	current := words[0]

	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
			continue
		}
		current += " " + word
	}

	lines = append(lines, current)
	return lines
}
