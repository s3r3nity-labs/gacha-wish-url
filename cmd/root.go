// Package cmd holds the Cobra commands.
package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Name is the program's name, also used in --version.
const Name = "gacha-wish-url"

// version is set at build time (-ldflags "-X .../cmd.version=v0.1.0"); otherwise it comes
// from the module version (go install ...@v0.1.0), and "dev" for a local build.
var version = ""

// Version returns the program version.
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// NewRootCmd builds the command tree. A function, not package globals, so every test gets
// fresh flags.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   Name + " <genshin|wuwa>",
		Short: "Copy the wish/convene history URL of Genshin Impact or Wuthering Waves",
		Long: `Finds the history URL in the game's cache (Genshin Impact) or log (Wuthering Waves),
prints it and copies it to the clipboard, for trackers like paimon.moe or wuwatracker.com.

Open the wish/convene history in the game first. The URL contains a temporary key to
your history: paste it into a tracker, don't share it elsewhere.`,
		Version:       Version(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate(Name + " {{.Version}}\n")
	for _, g := range games {
		root.AddCommand(newGameCmd(g))
	}
	return root
}
