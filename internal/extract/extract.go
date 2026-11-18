// Package extract pulls the history URL out of a game's cache or log file.
package extract

import (
	"regexp"
)

// LastMatch returns the last match of re in data, or "" if there is none. The files are
// partly binary (browser caches) and keep every URL the game ever opened; the last one
// is the newest, the one that still works.
func LastMatch(data []byte, re *regexp.Regexp) string {
	matches := re.FindAll(data, -1)
	if len(matches) == 0 {
		return ""
	}
	return string(matches[len(matches)-1])
}

// wuwaKey is the lookup table for Wuthering Waves' Client.log: every byte is XORed with
// 0xA5 if its low nibble is odd, else with 0xEF. The table is a one-to-one mapping but
// not its own inverse (0xA5 flips the lowest bit, so decoding again would pick the other
// key); it only decrypts.
var wuwaKey = func() (t [256]byte) {
	for b := range 256 {
		if (b&0x0F)%2 == 1 {
			t[b] = byte(b) ^ 0xA5
		} else {
			t[b] = byte(b) ^ 0xEF
		}
	}
	return t
}()

// DecryptWuWa decodes Wuthering Waves' Client.log, which newer game versions write
// scrambled.
func DecryptWuWa(data []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = wuwaKey[b]
	}
	return out
}
