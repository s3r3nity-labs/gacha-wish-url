# gacha-wish-url

Finds the wish/convene history URL of **Genshin Impact** and **Wuthering Waves** on Linux,
prints it and copies it to the clipboard, ready for trackers like
[paimon.moe](https://paimon.moe) or [wuwatracker.com](https://wuwatracker.com).

```fish
gacha-wish-url genshin   # open the wish history in the game first
gacha-wish-url wuwa      # keep the convene history open in the game while running this
```

> The URL contains a temporary key to your history. Paste it into a tracker; don't post
> it anywhere else.

## Installation

With [Homebrew](https://brew.sh) (e.g. on Bazzite), updated by `brew upgrade`:

```fish
brew install s3r3nity-labs/tap/gacha-wish-url
```

Or without Homebrew, from the latest release (one static binary):

```fish
curl -sL https://github.com/s3r3nity-labs/gacha-wish-url/releases/latest/download/gacha-wish-url_linux_amd64.tar.gz \
  | tar -xz -C ~/.local/bin gacha-wish-url
```

Works from any shell (fish, bash, zsh): it's a program, not a shell function.

## Where it looks

It searches the usual places and takes the most recently used file:

| Setup | Searched in |
| --- | --- |
| Steam (WuWa) | every Steam library from `libraryfolders.vdf`, also on other drives; native Steam and the Flatpak |
| Proton (e.g. Genshin added to Steam) | `steamapps/compatdata/*/pfx` of every library |
| Wine prefixes, Lutris, Heroic | `~/Games` |
| An Anime Game Launcher | `~/.local/share/anime-game-launcher`, and its Flatpak's data folder |
| Bottles | `~/.local/share/bottles/bottles`, and its Flatpak's data folder |

The files: `GenshinImpact_Data/webCaches/<version>/Cache/Cache_Data/data_2` (Genshin) and
`Client/Saved/Logs/Client.log` (WuWa, scrambled since 2.x; decrypted on the fly).

Not found? `-v` shows where it searched, and `--path` takes the file directly:

```fish
gacha-wish-url genshin -v
gacha-wish-url wuwa --path "/mnt/games/Wuthering Waves/Client/Saved/Logs/Client.log"
```

## Options

| Flag | |
| --- | --- |
| `-p`, `--print` | only print the URL, don't copy it |
| `--path FILE` | use this cache/log file instead of searching |
| `-v`, `--verbose` | show the search folders and every file found |

The URL goes to stdout, messages to stderr, so `gacha-wish-url wuwa -p | …` works.

Copying uses `wl-copy` (Wayland), `xclip` or `xsel` (X11), and else asks the terminal to
copy (OSC 52, supported by Konsole, Alacritty, kitty, WezTerm, foot and others).

## Development

```fish
go test ./...
go run . wuwa -v -p
```

- `cmd/` – the commands; `games.go` has what differs per game (URL pattern, files, hints)
- `internal/find/` – where to look (Steam libraries, prefixes, launchers)
- `internal/extract/` – the newest URL in a file, and WuWa's log decryption
- `internal/clip/` – clipboard with fallbacks

Releases: GitHub web UI → *Draft a new release* → new tag → *Publish*. GoReleaser builds
the binaries and updates the cask in
[s3r3nity-labs/homebrew-tap](https://github.com/s3r3nity-labs/homebrew-tap) (needs the
secret `HOMEBREW_TAP_GITHUB_TOKEN`).

Started 2025 as a fish function; rewritten in Go to work in every shell.
