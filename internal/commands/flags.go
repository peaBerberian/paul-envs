package commands

import (
	"flag"
	"strings"

	"github.com/peaberberian/paul-envs/internal/clihelp"
	"github.com/peaberberian/paul-envs/internal/console"
)

func newCommandFlagSet(name string, console *console.Console) *flag.FlagSet {
	flagset := flag.NewFlagSet(name, flag.ContinueOnError)
	flagset.SetOutput(console.Writer())
	return flagset
}

func parseCommandFlags(flagset *flag.FlagSet, args []string) error {
	var positional []string
	for len(args) > 0 {
		arg := args[0]
		if arg == "--" {
			positional = append(positional, args[1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			args = args[1:]
			continue
		}

		// A non-boolean flag consumes its next argument, even if it starts with '-'.
		name, _, hasValue := strings.Cut(strings.TrimPrefix(arg[1:], "-"), "=")
		n := 1
		if f := flagset.Lookup(name); f != nil && !hasValue {
			boolFlag, ok := f.Value.(interface{ IsBoolFlag() bool })
			if (!ok || !boolFlag.IsBoolFlag()) && len(args) > 1 {
				n = 2
			}
		}
		if err := flagset.Parse(args[:n]); err != nil {
			return err
		}
		args = args[n:]
	}
	return flagset.Parse(append([]string{"--"}, positional...))
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
	console.WriteLn("")
	console.WriteLn("Flags:")
	if hasFlags(flagset) {
		clihelp.PrintDefaults(console, flagset)
		console.WriteLn("")
	}
	console.WriteLn("  --help")
	console.WriteLn("    Show help.")
}

func hasFlags(flagset *flag.FlagSet) bool {
	hasAny := false
	flagset.VisitAll(func(*flag.Flag) {
		hasAny = true
	})
	return hasAny
}
