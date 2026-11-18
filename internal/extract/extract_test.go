package extract

import (
	"bytes"
	"regexp"
	"testing"
)

func TestLastMatchTakesTheNewestURL(t *testing.T) {
	// Printable ASCII only, non-greedy: never across binary bytes or into the next URL
	re := regexp.MustCompile(`https://example\.org/[!-~]*?end_id=0`)
	// A browser cache: binary noise around several URLs, the newest last
	data := []byte("\x00\x01junk https://example.org/old?authkey=A&end_id=0\x00\xff" +
		"more\x00https://example.org/new?authkey=B&end_id=0 \x02tail")

	if got := LastMatch(data, re); got != "https://example.org/new?authkey=B&end_id=0" {
		t.Errorf("got %q", got)
	}
	if got := LastMatch([]byte("nothing here"), re); got != "" {
		t.Errorf("no match: got %q", got)
	}
}

// encryptWuWa is the inverse of DecryptWuWa, to build test logs like the game writes them.
func encryptWuWa(plain []byte) []byte {
	var inv [256]byte
	for b := range 256 {
		inv[wuwaKey[b]] = byte(b)
	}
	out := make([]byte, len(plain))
	for i, b := range plain {
		out[i] = inv[b]
	}
	return out
}

func TestDecryptWuWa(t *testing.T) {
	plain := []byte(`[2026.11.18] LogTemp: Display: "url":"https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?svr_id=x&player_id=1&lang=en"`)
	scrambled := encryptWuWa(plain)
	if bytes.Equal(scrambled, plain) {
		t.Fatal("encryption did nothing")
	}
	if got := DecryptWuWa(scrambled); !bytes.Equal(got, plain) {
		t.Errorf("got %q", got)
	}
}

// The table must be a permutation, or some bytes couldn't be told apart after decoding.
func TestWuWaKeyIsOneToOne(t *testing.T) {
	seen := map[byte]bool{}
	for _, b := range wuwaKey {
		seen[b] = true
	}
	if len(seen) != 256 {
		t.Errorf("only %d distinct values", len(seen))
	}
}

// Same results as the Python version in the original fish function, for a few bytes.
func TestDecryptWuWaMatchesTheFishFunction(t *testing.T) {
	for in, want := range map[byte]byte{0x00: 0xEF, 0x01: 0xA4, 0x0F: 0xAA, 0xFE: 0x11, 0x41: 0xE4} {
		if got := DecryptWuWa([]byte{in})[0]; got != want {
			t.Errorf("%#02x -> %#02x, want %#02x", in, got, want)
		}
	}
}
