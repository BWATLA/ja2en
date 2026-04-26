// Package input resolves the active input source (args, stdin, clipboard,
// interactive multi-line).
package input

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
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

// readInteractiveStdin reads multi-line text from a TTY using a bubbletea v2
// textarea so the user gets a real editor: Backspace/Delete, Ctrl-A/E,
// Home/End, arrow keys, Ctrl-N/P (line up/down across multi-line buffer),
// word movement, etc. Enter inserts a newline, Ctrl-D submits, Ctrl-C
// aborts. The View() returns a tea.View whose Cursor field positions the
// real OS terminal cursor at the textarea's logical cursor — this is what
// lets CJK IME composition windows anchor to the input position instead of
// floating at the bottom of the terminal (the bubbletea v1 limitation that
// motivated the v2 migration). When stdin is piped (non-TTY), falls back to
// plain ReadAll so pipes still work in scripts/tests.
func readInteractiveStdin() (string, error) {
	if isStdinPiped() {
		// Interactive mode without a TTY is meaningless; defer to the
		// regular piped-stdin handling so `printf ... | ja2en -i` keeps
		// working in tests and scripts.
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

	m := newTextareaModel()
	final, err := tea.NewProgram(m, tea.WithOutput(os.Stderr)).Run()
	if err != nil {
		return "", fmt.Errorf("interactive editor: %w", err)
	}
	tm, ok := final.(textareaModel)
	if !ok {
		return "", fmt.Errorf("interactive editor: unexpected model type %T", final)
	}
	if tm.aborted {
		return "", fmt.Errorf("aborted")
	}
	text := strings.TrimSpace(tm.ta.Value())
	if text == "" {
		return "", fmt.Errorf("interactive input was empty")
	}
	return text, nil
}

// headerLines is the number of rows the helper text and the blank line below
// it consume above the textarea. Used to offset the textarea's reported
// cursor Y into screen coordinates.
const headerLines = 2

type textareaModel struct {
	ta        textarea.Model
	submitted bool
	aborted   bool
}

func newTextareaModel() textareaModel {
	ta := textarea.New()
	ta.Placeholder = "Enter Japanese text. Ctrl-D to translate, Ctrl-C to abort."
	ta.Prompt = "│ "
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // no limit
	ta.SetWidth(80)
	ta.SetHeight(8)
	// Disable the textarea's drawn block cursor so the OS terminal cursor
	// (positioned via View().Cursor below) is the only one visible. This
	// also makes CJK IMEs anchor composition windows to the right place.
	ta.SetVirtualCursor(false)
	ta.Focus()
	return textareaModel{ta: ta}
}

func (m textareaModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m textareaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case km.Mod&tea.ModCtrl != 0 && (km.Code == 'c' || km.Code == 'C'):
			m.aborted = true
			return m, tea.Quit
		case km.Mod&tea.ModCtrl != 0 && (km.Code == 'd' || km.Code == 'D'):
			m.submitted = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m textareaModel) View() tea.View {
	if m.submitted || m.aborted {
		return tea.NewView("")
	}
	content := "Enter Japanese text. Ctrl-D to translate, Ctrl-C to abort.\n\n" + m.ta.View() + "\n"

	v := tea.NewView(content)
	// Let the textarea report its own cursor position (handles prompt width,
	// CJK double-width runes, soft-wrap, padding, etc.), then offset by the
	// header rows we drew above it.
	if c := m.ta.Cursor(); c != nil {
		c.Y += headerLines
		v.Cursor = c
	}
	return v
}
