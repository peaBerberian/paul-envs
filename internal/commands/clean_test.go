package commands

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
)

func TestYesNoWithOptionalPrompt_NoPromptUsesDefault(t *testing.T) {
	out := &bytes.Buffer{}
	cons := console.New(context.Background(), strings.NewReader("n\n"), out, &bytes.Buffer{})

	got, err := yesNoWithOptionalPrompt(cons, true, "Remove cache?", false)
	if err != nil {
		t.Fatalf("yesNoWithOptionalPrompt() error = %v", err)
	}
	if got {
		t.Fatalf("yesNoWithOptionalPrompt() = %v, want false", got)
	}
	if !strings.Contains(out.String(), "Using default answer") {
		t.Fatalf("expected informational output, got %q", out.String())
	}
}
