package clihelp

import (
	"flag"
	"strings"

	"github.com/peaberberian/paul-envs/internal/console"
)

func PrintDefaults(console *console.Console, flagset *flag.FlagSet) {
	flagset.VisitAll(func(f *flag.Flag) {
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
		console.WriteLn("    %s", usage)
	})
}
