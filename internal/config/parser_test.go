package config

import (
	"strings"
	"testing"
)

// fakeHome is injected into parse() calls so tests don't depend on the real
// home directory of whoever runs them.
const fakeHome = "/home/testuser"

// helper builds a []Directive inline for comparison.
func dirs(pairs ...string) []Directive {
	if len(pairs)%2 != 0 {
		panic("dirs: odd number of arguments")
	}
	out := make([]Directive, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, Directive{Key: pairs[i], Value: pairs[i+1]})
	}
	return out
}

func parseString(s string) ([]Directive, error) {
	return parse(strings.NewReader(s), fakeHome)
}

func TestEmptyInput(t *testing.T) {
	got, err := parseString("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestSingleDirective(t *testing.T) {
	got, err := parseString("FROM ubuntu:24.04\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("FROM", "ubuntu:24.04")
	assertDirectives(t, want, got)
}

func TestValueWithInternalSpaces(t *testing.T) {
	// Everything after the first space is the value — including further spaces.
	got, err := parseString("LABEL my label has spaces\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("LABEL", "my label has spaces")
	assertDirectives(t, want, got)
}

func TestMultipleDirectives(t *testing.T) {
	input := "FROM ubuntu:24.04\nRUN apt-get update\nCMD /bin/bash\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs(
		"FROM", "ubuntu:24.04",
		"RUN", "apt-get update",
		"CMD", "/bin/bash",
	)
	assertDirectives(t, want, got)
}

func TestCommentLinesSkipped(t *testing.T) {
	input := "# this is a comment\nFROM alpine\n# another comment\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("FROM", "alpine")
	assertDirectives(t, want, got)
}

func TestCommentWithLeadingSpaces(t *testing.T) {
	// A line like "   # comment" should still be treated as a comment.
	input := "   # indented comment\nFROM alpine\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("FROM", "alpine")
	assertDirectives(t, want, got)
}

func TestInlineHashIsNotAComment(t *testing.T) {
	// A '#' that is NOT the first non-space character is part of the value.
	input := "ENV FOO=bar # not a comment\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("ENV", "FOO=bar # not a comment")
	assertDirectives(t, want, got)
}

func TestBlankLinesSkipped(t *testing.T) {
	input := "\nFROM alpine\n\n\nRUN echo hi\n\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("FROM", "alpine", "RUN", "echo hi")
	assertDirectives(t, want, got)
}

func TestWhitespaceOnlyLinesSkipped(t *testing.T) {
	input := "   \n\t\nFROM alpine\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("FROM", "alpine")
	assertDirectives(t, want, got)
}

func TestDuplicateDirectivesPreserved(t *testing.T) {
	input := "RUN apt-get update\nRUN apt-get install -y curl\nRUN apt-get clean\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs(
		"RUN", "apt-get update",
		"RUN", "apt-get install -y curl",
		"RUN", "apt-get clean",
	)
	assertDirectives(t, want, got)
}

func TestSameKeyDifferentValues(t *testing.T) {
	// Both ENV directives must survive; order matters.
	input := "ENV PATH=/usr/local/bin\nENV HOME=/root\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs(
		"ENV", "PATH=/usr/local/bin",
		"ENV", "HOME=/root",
	)
	assertDirectives(t, want, got)
}

func TestTildeSlashExpanded(t *testing.T) {
	// ~/foo/bar should become /home/testuser/foo/bar.
	input := "VOLUME ~/data\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("VOLUME", fakeHome+"/data")
	assertDirectives(t, want, got)
}

func TestTildeAloneExpanded(t *testing.T) {
	// A bare '~' value should expand to the home directory.
	input := "WORKDIR ~\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("WORKDIR", fakeHome)
	assertDirectives(t, want, got)
}

func TestTildeNotAtStartNotExpanded(t *testing.T) {
	// A '~' that is NOT the first character of the value must not be expanded.
	// Value is "foo~bar" — tilde is mid-string, no expansion should occur.
	input := "ENV foo~bar\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("ENV", "foo~bar")
	assertDirectives(t, want, got)
}

func TestTildeWithoutSlashNotExpanded(t *testing.T) {
	// '~something' (no slash) should NOT be expanded — it could be a username.
	input := "ENV ~otheruser\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be left as-is.
	want := dirs("ENV", "~otheruser")
	assertDirectives(t, want, got)
}

func TestTildeExpandedInMiddleOfDirectiveList(t *testing.T) {
	// Expansion works correctly for any position in the slice, not just last.
	input := "FROM alpine\nVOLUME ~/data\nRUN echo done\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs(
		"FROM", "alpine",
		"VOLUME", fakeHome+"/data",
		"RUN", "echo done",
	)
	assertDirectives(t, want, got)
}

func TestOrderPreserved(t *testing.T) {
	input := "Z last\nA first\nM middle\n"
	got, err := parseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("Z", "last", "A", "first", "M", "middle")
	assertDirectives(t, want, got)
}

func TestMissingValueReturnsError(t *testing.T) {
	// A non-blank, non-comment line with no space is malformed.
	_, err := parseString("NODIRECTIVE\n")
	if err == nil {
		t.Fatal("expected an error for a line without a space, got nil")
	}
}

func TestMissingValueAfterValidLineReturnsError(t *testing.T) {
	_, err := parseString("FROM alpine\nBADLINE\n")
	if err == nil {
		t.Fatal("expected an error for malformed line, got nil")
	}
}

func TestNoTrailingNewline(t *testing.T) {
	// Files that don't end with a newline are still parsed correctly.
	got, err := parseString("FROM alpine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dirs("FROM", "alpine")
	assertDirectives(t, want, got)
}

func assertDirectives(t *testing.T, want, got []Directive) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d directives, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("directive[%d]: want %+v, got %+v", i, want[i], got[i])
		}
	}
}
