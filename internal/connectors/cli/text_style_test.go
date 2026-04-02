package cli

import (
	"bytes"
	"testing"
)

func TestTextViewBuildsHealthOutput(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	view := newTextView(&out, false)

	got := view.
		Title("Billar Health").
		Divider("─────────────").
		Status("Status", "ok").
		Build()

	want := "Billar Health\n─────────────\nStatus: ok\n"
	if got != want {
		t.Fatalf("Build() = %q, want %q", got, want)
	}
}

func TestTextViewFieldAndLine(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	view := newTextView(&out, false)

	got := view.
		Field("Mode", "cli").
		Line("ready").
		Build()

	want := "Mode: cli\nready\n"
	if got != want {
		t.Fatalf("Build() = %q, want %q", got, want)
	}
}
