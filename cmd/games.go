package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/s3r3nity-labs/gacha-wish-url/internal/clip"
	"github.com/s3r3nity-labs/gacha-wish-url/internal/extract"
	"github.com/s3r3nity-labs/gacha-wish-url/internal/find"
)

// game is everything that differs between the games.
type game struct {
	cmd     string // subcommand
	title   string // e.g. "Genshin Impact"
	history string // what the game calls the history
	file    string // what's searched, for messages
	url     *regexp.Regexp
	find    func(home string) []find.File
	// decrypt: the file may be scrambled (Wuthering Waves' Client.log since 2.x)
	decrypt bool
	// keepOpen: the URL is only in the file while the history is open in the game
	keepOpen bool
}

// URL patterns: printable characters only and non-greedy, so a match never runs across
// the binary parts of a cache or into the next URL.
var games = []game{
	{
		cmd:     "genshin",
		title:   "Genshin Impact",
		history: "wish history",
		file:    "webCaches/*/Cache/Cache_Data/data_2",
		url:     regexp.MustCompile(`https://public-operation-hk4e-sg\.hoyoverse\.com[!-~]*?end_id=0`),
		find:    find.Genshin,
	},
	{
		cmd:      "wuwa",
		title:    "Wuthering Waves",
		history:  "convene history",
		file:     "Client/Saved/Logs/Client.log",
		url:      regexp.MustCompile(`https://aki-gm-resources-oversea\.aki-game\.net/aki/gacha/index\.html#/record[!#-~]*`),
		find:     find.WuWa,
		decrypt:  true,
		keepOpen: true,
	},
}

var (
	green  = color.New(color.Bold, color.FgGreen).SprintFunc()
	yellow = color.New(color.Bold, color.FgYellow).SprintFunc()
	faint  = color.New(color.Faint).SprintFunc()
)

func newGameCmd(g game) *cobra.Command {
	var path string
	var printOnly, verbose bool
	cmd := &cobra.Command{
		Use:   g.cmd,
		Short: fmt.Sprintf("Copy the %s %s URL", g.title, g.history),
		Long: fmt.Sprintf(`Finds the %s URL of %s (%s) and copies it to the clipboard.
Searches Steam (all libraries, also as a Flatpak), Proton and Wine prefixes, ~/Games
(Lutris, Heroic), An Anime Game Launcher and Bottles; -v shows where.`, g.history, g.title, g.file),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			url, err := g.run(cmd.ErrOrStderr(), home, path, verbose)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), url)
			if printOnly {
				return nil
			}
			return copyURL(cmd.ErrOrStderr(), g, url)
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "use this file instead of searching ("+g.file+")")
	cmd.Flags().BoolVarP(&printOnly, "print", "p", false, "only print the URL, don't copy it")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show where it searched and what it found")
	return cmd
}

// run finds the newest file and the newest URL in it.
func (g game) run(log io.Writer, home, path string, verbose bool) (string, error) {
	files, err := g.files(log, home, path, verbose)
	if err != nil {
		return "", err
	}
	newest := files[0].Path
	data, err := os.ReadFile(newest)
	if err != nil {
		return "", err
	}
	url := extract.LastMatch(data, g.url)
	if url == "" && g.decrypt {
		url = extract.LastMatch(extract.DecryptWuWa(data), g.url)
	}
	if url == "" {
		hint := fmt.Sprintf("Open the %s in the game, then run this again.", g.history)
		if g.keepOpen {
			hint = fmt.Sprintf("Open the %s in the game and keep it open while you run this.", g.history)
		}
		return "", fmt.Errorf("no %s URL in %s\n%s", g.history, newest, yellow(hint))
	}
	return url, nil
}

// files is --path, or the search result (newest first).
func (g game) files(log io.Writer, home, path string, verbose bool) ([]find.File, error) {
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		return []find.File{{Path: path, ModTime: info.ModTime()}}, nil
	}
	files := g.find(home)
	if verbose {
		fmt.Fprintln(log, faint("Searched:"))
		for _, r := range find.Roots(home) {
			state := "not there"
			if _, err := os.Stat(r.Path); err == nil {
				state = fmt.Sprintf("depth %d", r.Depth)
			}
			fmt.Fprintf(log, faint("  %s (%s)\n"), r.Path, state)
		}
		fmt.Fprintln(log, faint("Found:"))
		if len(files) == 0 {
			fmt.Fprintln(log, faint("  (nothing)"))
		}
		for i, f := range files {
			mark := " "
			if i == 0 {
				mark = "→"
			}
			fmt.Fprintf(log, faint("%s %s (%s)\n"), mark, f.Path, f.ModTime.Format(time.DateTime))
		}
	}
	if len(files) == 0 {
		return nil, errors.New(strings.Join([]string{
			fmt.Sprintf("no %s %s found", g.title, g.file),
			yellow(fmt.Sprintf("Start %s once, or give the file with --path; -v shows where it searched.", g.title)),
		}, "\n"))
	}
	return files, nil
}

// copyURL copies with whatever works here. Not being able to copy isn't an error: the URL
// is printed anyway.
func copyURL(log io.Writer, g game, url string) error {
	tty, _ := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if tty != nil {
		defer tty.Close()
	}
	var w io.Writer
	if tty != nil {
		w = tty
	}
	m, err := clip.System.Copy(url, w)
	switch {
	case err != nil:
		fmt.Fprintf(log, "\n%s copy the URL above yourself (%v)\n", yellow("Couldn't copy it:"), err)
	case m.Cmd == nil:
		fmt.Fprintf(log, "\n%s\n%s\n", green("✅ Sent the "+g.title+" "+g.history+" URL to your terminal's clipboard."),
			faint("If pasting doesn't work, your terminal doesn't support that (OSC 52): copy the URL above."))
	default:
		fmt.Fprintf(log, "\n%s %s\n", green("✅ Copied the "+g.title+" "+g.history+" URL to the clipboard."), faint("("+m.Name+")"))
	}
	return nil
}
