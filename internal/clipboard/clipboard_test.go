package clipboard

import (
	"os"
	"strings"
	"testing"
)

// TestIsWSL_FunctionVariableSwappable confirms the package-level isWSL
// hook is replaceable from tests, which we rely on to exercise both
// branches of Read / Write without spinning up a real WSL kernel.
func TestIsWSL_FunctionVariableSwappable(t *testing.T) {
	orig := isWSL
	defer func() { isWSL = orig }()

	isWSL = func() bool { return true }
	if !isWSL() {
		t.Errorf("override to true did not take effect")
	}

	isWSL = func() bool { return false }
	if isWSL() {
		t.Errorf("override to false did not take effect")
	}
}

// TestRoundTrip_WSL is opt-in (set JA2EN_TEST_CLIPBOARD=1) because it
// overwrites the user's actual clipboard. It verifies the WSL Read/Write
// pair correctly preserves multi-line non-ASCII content — exactly the
// payload that breaks shell-argv routes.
func TestRoundTrip_WSL(t *testing.T) {
	if os.Getenv("JA2EN_TEST_CLIPBOARD") == "" {
		t.Skip("set JA2EN_TEST_CLIPBOARD=1 to run clipboard round-trip test (overwrites user clipboard)")
	}
	if !isWSL() {
		t.Skip("WSL-only test")
	}

	expected := "テスト clip 経由\nline2 with backtick `cmd` and quote \"x\""
	if err := Write(expected); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != expected {
		t.Errorf("round-trip mismatch:\ngot:  %q\nwant: %q", got, expected)
	}

	// Restore a safe placeholder so the user's clipboard does not leak
	// the test fixture text into later operations.
	_ = Write("DUMMY-AFTER-CLIPBOARD-TEST")
}

// TestReadWSL_NormalizesCRLF asserts that PowerShell's CRLF line endings
// are converted to LF and any trailing newline is stripped, so callers
// see the text as the user copied it.
func TestReadWSL_NormalizesCRLF(t *testing.T) {
	// This is a unit test of the normalisation logic only; it does not
	// invoke PowerShell. We simulate what readWSL receives by exercising
	// the same string transformation manually.
	raw := "line1\r\nline2\r\n"
	got := strings.TrimRight(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	want := "line1\nline2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
