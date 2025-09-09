package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/cli"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/version"
)

func main() {
	filename, err := getFilename()
	if err != nil {
		log.Fatal(err)
	}

	es, err := evoke.NewStore(evoke.Config{DBFile: filename})
	if err != nil {
		log.Fatal(err)
	}
	defer es.Close()

	cs := commands.New(es)

	as := app.New(cs, es)

	kctx := kong.Parse(&cli.CLI)
	err = kctx.Run(as)
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
