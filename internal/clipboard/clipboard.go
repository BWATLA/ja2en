// Package clipboard wraps clipboard read/write operations.
//
// On WSL, atotto/clipboard's built-in backend probes for xclip / xsel /
// wl-copy / clip.exe / powershell.exe via PATH. In sandboxed shells (e.g.
// Claude Code's bubblewrap shell), powershell.exe and clip.exe are not on
// PATH even though their files exist under /mnt/c/, so the probe fails and
// reads return "exit status 1". To make `--clip` and `--paste` work
// reliably in every WSL shell variant, we detect WSL via /proc/version and
// invoke powershell.exe / clip.exe directly via their hardcoded full paths.
//
// On native Linux / macOS we keep using atotto/clipboard so its built-in
// backend selection (xclip / xsel / pbpaste / wl-copy) continues to work.
package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
)

// Hardcoded WSL paths to the Windows-native clipboard helpers. These are
// stable on every supported Windows install — the system32 directory is
// not user-relocatable.
const (
	wslPowerShellPath = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
	wslClipPath       = "/mnt/c/Windows/System32/clip.exe"
)

// isWSL reports whether the current process is running under WSL,
// detected by inspecting /proc/version (which contains "Microsoft" or
// "WSL" on every WSL kernel build).
var isWSL = func() bool {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	s := strings.ToLower(string(b))
	return strings.Contains(s, "microsoft") || strings.Contains(s, "wsl")
}

// Read returns the current clipboard text.
func Read() (string, error) {
	if isWSL() {
		return readWSL()
	}
	return clipboard.ReadAll()
}

func readWSL() (string, error) {
	cmd := exec.Command(wslPowerShellPath, "-NoProfile", "-Command", "Get-Clipboard")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("powershell Get-Clipboard at %s: %w", wslPowerShellPath, err)
	}
	// PowerShell emits CRLF and a trailing newline; normalise to LF and
	// strip the trailing terminator so callers see the same text the user
	// copied.
	s := strings.ReplaceAll(string(out), "\r\n", "\n")
	s = strings.TrimRight(s, "\n")
	return s, nil
}

// Write replaces the clipboard with the given text.
func Write(text string) error {
	if isWSL() {
		return writeWSL(text)
	}
	return clipboard.WriteAll(text)
}

func writeWSL(text string) error {
	cmd := exec.Command(wslClipPath)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clip.exe at %s: %w", wslClipPath, err)
	}
	return nil
}
