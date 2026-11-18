package clip

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fake(env map[string]string, tools ...string) Env {
	return Env{
		Getenv: func(k string) string { return env[k] },
		LookPath: func(name string) (string, error) {
			for _, t := range tools {
				if t == name {
					return "/usr/bin/" + name, nil
				}
			}
			return "", errors.New("not found")
		},
	}
}

func names(ms []Method) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return out
}

func TestMethods(t *testing.T) {
	tests := []struct {
		name string
		env  Env
		want []string
	}{
		{"wayland", fake(map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"}, "wl-copy", "xclip"),
			[]string{"wl-copy", "xclip", "terminal (OSC 52)"}},
		{"wayland without wl-copy (Bazzite without wl-clipboard)", fake(map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"}, "xsel"),
			[]string{"xsel", "terminal (OSC 52)"}},
		{"x11", fake(map[string]string{"DISPLAY": ":0"}, "wl-copy", "xclip", "xsel"),
			[]string{"xclip", "xsel", "terminal (OSC 52)"}},
		{"ssh, no display", fake(map[string]string{}, "wl-copy", "xclip"),
			[]string{"terminal (OSC 52)"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := names(tt.env.Methods())
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCopyFallsBackToTheTerminal(t *testing.T) {
	var tty bytes.Buffer
	m, err := fake(map[string]string{}).Copy("https://example.org/x", &tty)
	if err != nil || m.Name != "terminal (OSC 52)" {
		t.Fatalf("method %q, err %v", m.Name, err)
	}
	if tty.String() != "\x1b]52;c;aHR0cHM6Ly9leGFtcGxlLm9yZy94\a" {
		t.Errorf("wrote %q", tty.String())
	}
}

func TestCopyWithNothingAvailable(t *testing.T) {
	if _, err := fake(map[string]string{}).Copy("x", nil); err == nil {
		t.Error("no error without any way to copy")
	}
}

// wl-copy forks a process that keeps the clipboard alive and inherits stdout/stderr.
// run must return when the program itself exits, not wait for that process.
func TestRunDoesNotWaitForTheProcessKeepingTheClipboard(t *testing.T) {
	got := filepath.Join(t.TempDir(), "clipboard")
	start := time.Now()

	err := run([]string{"sh", "-c", `cat > "$0"; sleep 5 &`, got}, "hello")

	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("waited %v for the background process", d)
	}
	if data, _ := os.ReadFile(got); string(data) != "hello" {
		t.Errorf("program got %q on stdin", data)
	}
}

func TestRunReportsFailure(t *testing.T) {
	if err := run([]string{"sh", "-c", "exit 3"}, "x"); err == nil {
		t.Error("exit status 3 not reported")
	}
}
