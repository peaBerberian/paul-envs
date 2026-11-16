package commands

import (
	"flag"
	"io"
	"slices"
	"testing"
)

func TestParseCommandFlags(t *testing.T) {
	flags := flag.NewFlagSet("stop", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	engine := flags.String("engine", "", "")
	timeout := flags.Int("timeout", 0, "")
	err := parseCommandFlags(flags, []string{"--engine", "docker", "alpha", "--timeout", "-1", "--", "--help"})
	if err != nil {
		t.Fatal(err)
	}
	if *engine != "docker" || *timeout != -1 || !slices.Equal(flags.Args(), []string{"alpha", "--help"}) {
		t.Fatalf("engine=%q timeout=%d args=%q", *engine, *timeout, flags.Args())
	}
	var visited []string
	flags.Visit(func(f *flag.Flag) { visited = append(visited, f.Name) })
	if !slices.Equal(visited, []string{"engine", "timeout"}) {
		t.Fatalf("explicit flags = %q", visited)
	}
}

func TestParseRunFlags(t *testing.T) {
	for _, command := range [][]string{
		{"alpha", "echo", "--help"},
		{"alpha", "echo", "--engine", "podman"},
		{"alpha", "echo", "--", "-n"},
	} {
		t.Run(command[2], func(t *testing.T) {
			flags := flag.NewFlagSet("run", flag.ContinueOnError)
			flags.SetOutput(io.Discard)
			engine := flags.String("engine", "", "")
			args := append([]string{"--engine", "docker"}, command...)
			if err := parseRunFlags(flags, args); err != nil {
				t.Fatal(err)
			}
			if *engine != "docker" || !slices.Equal(flags.Args(), command) {
				t.Fatalf("engine=%q args=%q, want docker and %q", *engine, flags.Args(), command)
			}
		})
	}
}
