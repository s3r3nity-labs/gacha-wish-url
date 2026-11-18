// gacha-wish-url copies the wish/convene history URL of Genshin Impact or Wuthering Waves.
package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/s3r3nity-labs/gacha-wish-url/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.New(color.Bold, color.FgRed).Sprint("❌ ")+err.Error())
		os.Exit(1)
	}
}
