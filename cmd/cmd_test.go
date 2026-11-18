package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func init() { color.NoColor = true }

func run(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func write(t *testing.T, path string, data []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const (
	oldGenshin = "https://public-operation-hk4e-sg.hoyoverse.com/gacha_info/api/getGachaLog?authkey=OLD&end_id=0"
	newGenshin = "https://public-operation-hk4e-sg.hoyoverse.com/gacha_info/api/getGachaLog?authkey=NEW&end_id=0"
	wuwaURL    = "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?svr_id=abc&player_id=123&lang=en&gacha_id=1&record_id=xyz"
)

func TestGenshinPrintsTheNewestURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	write(t, filepath.Join(home, "Games", "genshin-impact", "drive_c", "Program Files", "HoYoPlay", "games",
		"Genshin Impact game", "GenshinImpact_Data", "webCaches", "2.38.0.0", "Cache", "Cache_Data", "data_2"),
		[]byte("\x00\x10"+oldGenshin+"\x00\xff\x01junk\x00"+newGenshin+"\x00\x00"))

	out, _, err := run(t, "genshin", "--print")

	if err != nil || out != newGenshin+"\n" {
		t.Fatalf("out %q, err %v", out, err)
	}
}

func TestWuWaDecryptsTheLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	plain := []byte(`[2026.11.18-20.00.00:000][  0]LogKuro: Display: open url "` + wuwaURL + `" done`)
	// scramble it the way the game does (inverse of the decryption table)
	scrambled := make([]byte, len(plain))
	for i, b := range plain {
		for c := range 256 {
			if c^map[bool]int{true: 0xA5, false: 0xEF}[(c&0x0F)%2 == 1] == int(b) {
				scrambled[i] = byte(c)
				break
			}
		}
	}
	write(t, filepath.Join(home, ".local", "share", "Steam", "steamapps", "common", "Wuthering Waves",
		"Client", "Saved", "Logs", "Client.log"), scrambled)

	out, _, err := run(t, "wuwa", "-p")

	if err != nil || out != wuwaURL+"\n" {
		t.Fatalf("out %q, err %v", out, err)
	}
}

func TestWuWaPlainLogWorksToo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	log := write(t, filepath.Join(t.TempDir(), "Client.log"), []byte(`url "`+wuwaURL+`"`))

	out, _, err := run(t, "wuwa", "--print", "--path", log)

	if err != nil || out != wuwaURL+"\n" {
		t.Fatalf("out %q, err %v", out, err)
	}
}

func TestNoURLAsksToOpenTheHistory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	log := write(t, filepath.Join(t.TempDir(), "Client.log"), []byte("nothing useful"))

	_, _, err := run(t, "wuwa", "--print", "--path", log)

	if err == nil || !strings.Contains(err.Error(), "keep it open") {
		t.Fatalf("err %v", err)
	}
}

func TestNotInstalledExplainsWhatToDo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, stderr, err := run(t, "genshin", "-p", "-v")

	if err == nil || !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err %v", err)
	}
	if !strings.Contains(stderr, "Searched:") || !strings.Contains(stderr, "Games") {
		t.Errorf("verbose output:\n%s", stderr)
	}
}

func TestVersionAndUnknownGame(t *testing.T) {
	version = "v1.2.3"
	t.Cleanup(func() { version = "" })
	if out, _, err := run(t, "--version"); err != nil || out != Name+" v1.2.3\n" {
		t.Errorf("version: %q %v", out, err)
	}
	if _, _, err := run(t, "honkai"); err == nil {
		t.Error("unknown game accepted")
	}
}
