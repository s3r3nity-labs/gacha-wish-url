// Package clip copies text to the clipboard on Linux, with whatever is there: wl-copy
// (Wayland), xclip or xsel (X11), or else OSC 52, an escape sequence that asks the
// terminal itself to copy (Konsole, Alacritty, kitty, WezTerm, foot, …).
package clip

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Method is one way to copy.
type Method struct {
	Name string   // shown to the user, e.g. "wl-copy"
	Cmd  []string // the command, text on stdin; nil for OSC 52
}

// Env is what the choice depends on, so tests can fake it.
type Env struct {
	Getenv   func(string) string
	LookPath func(string) (string, error)
}

// System is the real environment.
var System = Env{Getenv: os.Getenv, LookPath: exec.LookPath}

// Methods lists the ways to copy that should work here, best first. OSC 52 is always
// last: it can't tell whether the terminal did it.
func (e Env) Methods() []Method {
	var out []Method
	have := func(name string) bool { _, err := e.LookPath(name); return err == nil }
	if e.Getenv("WAYLAND_DISPLAY") != "" && have("wl-copy") {
		out = append(out, Method{"wl-copy", []string{"wl-copy"}})
	}
	if e.Getenv("DISPLAY") != "" {
		if have("xclip") {
			out = append(out, Method{"xclip", []string{"xclip", "-selection", "clipboard"}})
		}
		if have("xsel") {
			out = append(out, Method{"xsel", []string{"xsel", "--clipboard", "--input"}})
		}
	}
	return append(out, Method{Name: "terminal (OSC 52)"})
}

// Copy tries the methods in order and returns the one that worked. tty is where OSC 52
// is written (the terminal); nil skips it.
func (e Env) Copy(text string, tty io.Writer) (Method, error) {
	var errs []error
	for _, m := range e.Methods() {
		if m.Cmd == nil {
			if tty == nil {
				continue
			}
			if _, err := io.WriteString(tty, OSC52(text)); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", m.Name, err))
				continue
			}
			return m, nil
		}
		if err := run(m.Cmd, text); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", m.Name, err))
			continue
		}
		return m, nil
	}
	if len(errs) == 0 {
		errs = append(errs, errors.New("no clipboard tool (wl-copy, xclip, xsel) and no terminal"))
	}
	return Method{}, errors.Join(errs...)
}

// run starts a clipboard program with text on stdin and waits until it exits. Its output
// goes to /dev/null on purpose: wl-copy and xclip leave a process behind that keeps the
// clipboard alive, and it inherits stdout/stderr. Capturing the output would wait for
// that process to close them, i.e. until something else is copied (the tool would hang).
func run(command []string, text string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout = nil // /dev/null
	cmd.Stderr = nil
	return cmd.Run()
}

// OSC52 is the escape sequence asking the terminal to put text into the clipboard.
func OSC52(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\a"
}
