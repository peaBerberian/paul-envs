package clihelp

import (
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
)

func TestPrintDefaultsWrapsParagraphsAndSeparatesFlags(t *testing.T) {
	var out strings.Builder
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	flagset.String("alpha", "", "First sentence that should wrap cleanly when it exceeds the target width for the help output.\nSecond paragraph with extra details.")
	flagset.Bool("beta", false, "Short description.")

	PrintDefaults(cons, flagset)

	got := out.String()
	wantFragments := []string{
		"--alpha string",
		"    First sentence that should wrap cleanly when it exceeds the target width for",
		"    the help output.",
		"    Second paragraph with extra details.",
		"--beta",
		"    Short description.",
	}

	for _, fragment := range wantFragments {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%s", fragment, got)
		}
	}
}
