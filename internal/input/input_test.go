package input

import (
	"os"
	"testing"
)

func TestReadInteractiveStdin_PipedFallthrough(t *testing.T) {
	// When stdin is piped (which is what os.Pipe gives us in a test),
	// readInteractiveStdin must fall back to io.ReadAll without printing
	// the "Enter Japanese text..." prompt.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	expected := "改行\n含む\nテキスト"
	if _, err := w.Write([]byte(expected)); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	got, err := readInteractiveStdin()
	if err != nil {
		t.Fatalf("readInteractiveStdin: %v", err)
	}
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestResolve_InteractiveWithPipedStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	expected := "改行ありの\n対話モード入力"
	if _, err := w.Write([]byte(expected + "\n")); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	got, err := Resolve(Source{UseInteractive: true})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
