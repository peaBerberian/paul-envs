package commands

import (
	"flag"

	"github.com/peaberberian/paul-envs/internal/clihelp"
	"github.com/peaberberian/paul-envs/internal/console"
)

func newCommandFlagSet(name string, console *console.Console) *flag.FlagSet {
	flagset := flag.NewFlagSet(name, flag.ContinueOnError)
	flagset.SetOutput(console.Writer())
	return flagset
}

func parseCommandFlags(flagset *flag.FlagSet, args []string) error {
	return flagset.Parse(args)
}

func writeCommandUsage(
	console *console.Console,
	flagset *flag.FlagSet,
	usageLine string,
	description string,
) {
	console.WriteLn("Usage: %s", usageLine)
	if description != "" {
		console.WriteLn("")
		console.WriteLn("%s", description)
	}
	if hasFlags(flagset) {
		console.WriteLn("")
		console.WriteLn("Flags:")
		clihelp.PrintDefaults(console, flagset)
	}
}

func hasFlags(flagset *flag.FlagSet) bool {
	hasAny := false
	flagset.VisitAll(func(*flag.Flag) {
		hasAny = true
	})
	return hasAny
}
