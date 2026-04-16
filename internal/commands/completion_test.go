package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
)

func TestCompletionHelp(t *testing.T) {
	var out strings.Builder
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := Completion(context.Background(), []string{"--help"}, cons); err != nil {
		t.Fatalf("Completion(--help) error = %v", err)
	}

	got := out.String()
	wantFragments := []string{
		"Usage: paul-envs completion <bash|zsh|fish> [flags]",
		"Print the shell completion script for the selected shell to standard output.",
		"--help",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%s", fragment, got)
		}
	}
}

func TestCompletionRequiresShell(t *testing.T) {
	var out strings.Builder
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	err := Completion(context.Background(), nil, cons)
	if err == nil {
		t.Fatal("Completion() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "expected exactly one shell argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionRejectsUnknownShell(t *testing.T) {
	var out strings.Builder
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	err := Completion(context.Background(), []string{"tcsh"}, cons)
	if err == nil {
		t.Fatal("Completion(tcsh) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid shell") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionOutputsRawScript(t *testing.T) {
	var out strings.Builder
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := Completion(context.Background(), []string{"bash"}, cons); err != nil {
		t.Fatalf("Completion(bash) error = %v", err)
	}

	got := out.String()
	if !strings.HasPrefix(got, "_paulenvs()") {
		t.Fatalf("unexpected bash completion output:\n%s", got)
	}
	if !strings.Contains(got, "complete -F _paulenvs paul-envs") {
		t.Fatalf("missing completion registration in output:\n%s", got)
	}
}
