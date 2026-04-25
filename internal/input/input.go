// Package input resolves the active input source (args, stdin, clipboard,
// interactive multi-line).
package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GigiTiti-Kai/ja2en/internal/clipboard"
)

// Source describes which input channels are enabled by the caller.
type Source struct {
	Args           []string
	UseClip        bool
	UseInteractive bool
}

// Resolve picks the active input. Precedence (highest to lowest):
//  1. --interactive flag → multi-line stdin until EOF (Ctrl-D)
//  2. --clip explicit flag → clipboard
//  3. positional args      → args joined by space
//  4. piped stdin          → all of stdin
//
// An empty/whitespace-only result yields an error so callers can stop early.
func Resolve(s Source) (string, error) {
	if s.UseInteractive {
		return readInteractiveStdin()
	}

	if s.UseClip {
		text, err := clipboard.Read()
		if err != nil {
			return "", fmt.Errorf("read clipboard: %w", err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return "", fmt.Errorf("clipboard is empty")
		}
		return text, nil
	}

	if len(s.Args) > 0 {
		text := strings.TrimSpace(strings.Join(s.Args, " "))
		if text != "" {
			return text, nil
		}
	}

	if isStdinPiped() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		text := strings.TrimSpace(string(data))
		if text != "" {
			return text, nil
		}
	}

	return "", fmt.Errorf("no input. pass text as argument, pipe to stdin, or use --clip / --interactive")
}

func isStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// readInteractiveStdin reads multi-line text from stdin until EOF (Ctrl-D).
// Useful when the user wants to paste or type free-form Japanese including
// newlines and shell-special characters that would otherwise be mangled by
// argv parsing.
func readInteractiveStdin() (string, error) {
	if isStdinPiped() {
		// If stdin is already piped, defer to the regular piped-stdin
		// handling — interactive mode without a TTY is meaningless.
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return "", fmt.Errorf("interactive input was empty")
		}
		return text, nil
	}

	fmt.Fprintln(os.Stderr, "Enter Japanese text. Ctrl-D to translate, Ctrl-C to abort.")
	var buf strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	// Bump max token size from 64 KiB to 1 MiB so a single line of pasted
	// content (e.g. a long prose paragraph) does not trip ErrTooLong.
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	text := strings.TrimSpace(buf.String())
	if text == "" {
		return "", fmt.Errorf("interactive input was empty")
	}
	return text, nil
}
