// Package find looks for the games' cache and log files on Linux, wherever they're
// installed: Steam (also on other library drives and as a Flatpak), Proton prefixes,
// Wine prefixes in ~/Games (Lutris, Heroic, own), An Anime Game Launcher and Bottles.
package find

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// File is a found cache or log file.
type File struct {
	Path    string
	ModTime time.Time
}

// Root is where a search starts and how deep below it a game folder may be.
type Root struct {
	Path  string
	Depth int
}

// prefixDepth is enough for game folders inside Wine/Proton prefixes, e.g.
// compatdata/<id>/pfx/drive_c/Program Files/HoYoPlay/games/Genshin Impact game/
// GenshinImpact_Data is 8 below steamapps/compatdata.
const prefixDepth = 9

// Genshin finds Genshin Impact's browser cache files, newest first:
// …/GenshinImpact_Data/webCaches/<version>/Cache/Cache_Data/data_2.
func Genshin(home string) []File {
	return search(roots(home), func(dir, name string) ([]string, bool) {
		if name != "GenshinImpact_Data" {
			return nil, false
		}
		found, _ := filepath.Glob(filepath.Join(dir, "webCaches", "*", "Cache", "Cache_Data", "data_2"))
		return found, true
	})
}

// WuWa finds Wuthering Waves' client logs, newest first:
// …/Wuthering Waves[ Game]/Client/Saved/Logs/Client.log.
func WuWa(home string) []File {
	return search(roots(home), func(dir, name string) ([]string, bool) {
		if !strings.HasPrefix(name, "Wuthering Waves") {
			return nil, false
		}
		// Steam: "Wuthering Waves/Client", Kuro's launcher: "Wuthering Waves Game/Client",
		// sometimes one folder deeper
		var found []string
		for _, pattern := range []string{
			filepath.Join(dir, "Client", "Saved", "Logs", "Client.log"),
			filepath.Join(dir, "*", "Client", "Saved", "Logs", "Client.log"),
		} {
			m, _ := filepath.Glob(pattern)
			found = append(found, m...)
		}
		return found, len(found) > 0
	})
}

// Roots lists where the search starts, for --verbose.
func Roots(home string) []Root {
	return roots(home)
}

func roots(home string) []Root {
	var out []Root
	for _, p := range []string{
		filepath.Join(home, "Games"), // own Wine prefixes, Lutris, Heroic
		filepath.Join(home, ".local", "share", "anime-game-launcher"),
		filepath.Join(home, ".var", "app", "moe.launcher.an-anime-game-launcher", "data", "anime-game-launcher"),
		filepath.Join(home, ".local", "share", "bottles", "bottles"),
		filepath.Join(home, ".var", "app", "com.usebottles.bottles", "data", "bottles", "bottles"),
	} {
		out = append(out, Root{p, prefixDepth})
	}
	for _, lib := range SteamLibraries(home) {
		out = append(out,
			// Steam games sit directly in common (WuWa); don't walk through every game
			Root{filepath.Join(lib, "steamapps", "common"), 1},
			// Proton prefixes, e.g. Genshin added to Steam as a non-Steam game
			Root{filepath.Join(lib, "steamapps", "compatdata"), prefixDepth},
		)
	}
	seen := map[string]bool{}
	var unique []Root
	for _, r := range out {
		key := realPath(r.Path)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, r)
		}
	}
	return unique
}

// steamRoots are where Steam lives: native (two usual paths, often one a link to the
// other) and as a Flatpak.
func steamRoots(home string) []string {
	return []string{
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"),
	}
}

var vdfPath = regexp.MustCompile(`"path"\s+"([^"]+)"`)

// SteamLibraries lists every Steam library folder: each Steam installation itself, plus
// the folders in its libraryfolders.vdf (other drives).
func SteamLibraries(home string) []string {
	var libs []string
	for _, root := range steamRoots(home) {
		if !isDir(root) {
			continue
		}
		libs = append(libs, root)
		for _, vdf := range []string{
			filepath.Join(root, "steamapps", "libraryfolders.vdf"),
			filepath.Join(root, "config", "libraryfolders.vdf"),
		} {
			data, err := os.ReadFile(vdf)
			if err != nil {
				continue
			}
			for _, m := range vdfPath.FindAllSubmatch(data, -1) {
				// VDF escapes backslashes; Linux paths have none, but be safe
				libs = append(libs, strings.ReplaceAll(string(m[1]), `\\`, `\`))
			}
		}
	}
	return dedupe(libs)
}

// skipDirs are never worth descending into: Wine's system folder, other games' Unity data,
// caches of other programs.
func skipDir(name string) bool {
	switch name {
	case "windows", "dosdevices", "shadercache", "node_modules", ".git", "Engine", "Binaries":
		return true
	}
	return strings.HasSuffix(name, "_Data") && name != "GenshinImpact_Data"
}

// search walks every root up to its depth. match is called for each folder: it returns the
// files found there, and whether to stop descending into it. Symlinks aren't followed
// (Wine's z: drive points to /). Results are newest first.
func search(roots []Root, match func(dir, name string) ([]string, bool)) []File {
	seen := map[string]bool{}
	var out []File
	for _, r := range roots {
		root := r.Path
		if !isDir(root) {
			continue
		}
		base := strings.Count(filepath.Clean(root), string(filepath.Separator))
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable folder: skip it, keep going
			}
			if !d.IsDir() {
				return nil
			}
			if path != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			files, stop := match(path, d.Name())
			for _, f := range files {
				real, err := filepath.EvalSymlinks(f)
				if err != nil || seen[real] {
					continue
				}
				seen[real] = true
				if info, err := os.Stat(f); err == nil && info.Mode().IsRegular() {
					out = append(out, File{Path: f, ModTime: info.ModTime()})
				}
			}
			if stop || strings.Count(path, string(filepath.Separator))-base >= r.Depth {
				return filepath.SkipDir
			}
			return nil
		})
	}
	slices.SortStableFunc(out, func(a, b File) int { return b.ModTime.Compare(a.ModTime) })
	return out
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// realPath resolves links, so two paths to the same folder count once
// (~/.steam/steam is usually a link to ~/.local/share/Steam).
func realPath(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}

// dedupe drops repeated folders, also when two paths lead to the same one.
func dedupe(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		if key := realPath(p); !seen[key] {
			seen[key] = true
			out = append(out, p)
		}
	}
	return out
}
