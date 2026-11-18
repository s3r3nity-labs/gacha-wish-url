package find

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// file creates a file (and its folders) with a modification time some minutes ago.
func file(t *testing.T, path string, minutesAgo int) string {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0o755))
	must(t, os.WriteFile(path, []byte("x"), 0o644))
	at := time.Now().Add(-time.Duration(minutesAgo) * time.Minute)
	must(t, os.Chtimes(path, at, at))
	return path
}

func paths(files []File) []string {
	var out []string
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

const genshinTail = "Program Files/HoYoPlay/games/Genshin Impact game/GenshinImpact_Data/webCaches"

func TestGenshinOwnWinePrefixNewestVersionFirst(t *testing.T) {
	home := t.TempDir()
	prefix := filepath.Join(home, "Games", "genshin-impact", "drive_c")
	old := file(t, filepath.Join(prefix, genshinTail, "2.37.0.0", "Cache", "Cache_Data", "data_2"), 60)
	cur := file(t, filepath.Join(prefix, genshinTail, "2.38.0.0", "Cache", "Cache_Data", "data_2"), 1)

	got := paths(Genshin(home))

	if len(got) != 2 || got[0] != cur || got[1] != old {
		t.Errorf("got %v", got)
	}
}

func TestGenshinAnimeGameLauncherFlatpak(t *testing.T) {
	home := t.TempDir()
	want := file(t, filepath.Join(home, ".var", "app", "moe.launcher.an-anime-game-launcher", "data",
		"anime-game-launcher", "Genshin Impact", "GenshinImpact_Data", "webCaches", "2.38.0.0",
		"Cache", "Cache_Data", "data_2"), 1)

	if got := paths(Genshin(home)); len(got) != 1 || got[0] != want {
		t.Errorf("got %v", got)
	}
}

func TestGenshinInAProtonPrefix(t *testing.T) {
	home := t.TempDir()
	want := file(t, filepath.Join(home, ".local", "share", "Steam", "steamapps", "compatdata", "2873049152",
		"pfx", "drive_c", genshinTail, "2.38.0.0", "Cache", "Cache_Data", "data_2"), 1)

	if got := paths(Genshin(home)); len(got) != 1 || got[0] != want {
		t.Errorf("got %v", got)
	}
}

func TestWuWaOnASecondSteamLibrary(t *testing.T) {
	home := t.TempDir()
	drive := t.TempDir() // e.g. /run/media/kiwi/Games
	steam := filepath.Join(home, ".local", "share", "Steam")
	must(t, os.MkdirAll(filepath.Join(steam, "steamapps"), 0o755))
	// ~/.steam/steam is a link to the same Steam: must not be found twice
	must(t, os.MkdirAll(filepath.Join(home, ".steam"), 0o755))
	must(t, os.Symlink(steam, filepath.Join(home, ".steam", "steam")))
	must(t, os.WriteFile(filepath.Join(steam, "steamapps", "libraryfolders.vdf"), []byte(`"libraryfolders"
{
	"0"
	{
		"path"		"`+steam+`"
		"label"		""
	}
	"1"
	{
		"path"		"`+drive+`/SteamLibrary"
		"apps"		{ "3513350"		"123" }
	}
}`), 0o644))
	want := file(t, filepath.Join(drive, "SteamLibrary", "steamapps", "common", "Wuthering Waves",
		"Client", "Saved", "Logs", "Client.log"), 1)

	libs := SteamLibraries(home)
	if len(libs) != 2 {
		t.Errorf("libraries %v", libs)
	}
	if got := paths(WuWa(home)); len(got) != 1 || got[0] != want {
		t.Errorf("got %v", got)
	}
}

func TestWuWaFlatpakSteamAndKuroLauncher(t *testing.T) {
	home := t.TempDir()
	steamWuWa := file(t, filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam",
		"steamapps", "common", "Wuthering Waves", "Client", "Saved", "Logs", "Client.log"), 30)
	// Kuro's own launcher in a Wine prefix: "Wuthering Waves Game"
	launcher := file(t, filepath.Join(home, "Games", "wuwa", "drive_c", "Wuthering Waves",
		"Wuthering Waves Game", "Client", "Saved", "Logs", "Client.log"), 2)

	got := paths(WuWa(home))
	if len(got) != 2 || got[0] != launcher || got[1] != steamWuWa {
		t.Errorf("got %v", got)
	}
}

// Wine's z: drive is a link to /. Following it would walk the whole system (and find
// everything twice); windows/ is never searched.
func TestDoesNotFollowWineLinksOrSearchWindows(t *testing.T) {
	home := t.TempDir()
	prefix := filepath.Join(home, "Games", "genshin-impact")
	must(t, os.MkdirAll(filepath.Join(prefix, "dosdevices"), 0o755))
	must(t, os.Symlink("/", filepath.Join(prefix, "dosdevices", "z:")))
	must(t, os.Symlink(home, filepath.Join(prefix, "drive_c_link")))
	file(t, filepath.Join(prefix, "drive_c", "windows", "GenshinImpact_Data", "webCaches", "1", "Cache", "Cache_Data", "data_2"), 1)
	want := file(t, filepath.Join(prefix, "drive_c", genshinTail, "2.38.0.0", "Cache", "Cache_Data", "data_2"), 1)

	start := time.Now()
	got := paths(Genshin(home))
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %v", got)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("took %v: followed a link?", time.Since(start))
	}
}

func TestOnlyWalksDirectlyIntoSteamGames(t *testing.T) {
	home := t.TempDir()
	common := filepath.Join(home, ".local", "share", "Steam", "steamapps", "common")
	// A WuWa folder deep inside another game is not WuWa
	file(t, filepath.Join(common, "Other Game", "Mods", "Wuthering Waves", "Client", "Saved", "Logs", "Client.log"), 1)

	if got := WuWa(home); len(got) != 0 {
		t.Errorf("searched inside other games: %v", paths(got))
	}
}

func TestNothingInstalled(t *testing.T) {
	home := t.TempDir()
	if g, w := Genshin(home), WuWa(home); len(g)+len(w) != 0 {
		t.Errorf("found %v %v", g, w)
	}
	for _, r := range Roots(home) {
		if !strings.HasPrefix(r.Path, home) {
			t.Errorf("root outside home: %s", r.Path)
		}
	}
}
