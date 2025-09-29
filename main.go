package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/cli"
	"github.com/rcy/whatever/version"
)

func main() {
	a := app.New()

	kctx := kong.Parse(&cli.CLI)
	err := kctx.Run(a)
	kctx.FatalIfErrorf(err)
}

func getFilename() (string, error) {
	if os.Getenv("FILENAME") != "" {
		return os.Getenv("FILENAME"), nil
	}
	base, _ := os.UserConfigDir()
	filename := base + "/whatever/flog.sqlite"
	if !version.IsRelease() {
		filename = base + "/whatever-dev/flog.sqlite"
	}
	return filename, nil
}
