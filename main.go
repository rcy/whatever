package main

import (
	"log"
	"os"

	_ "modernc.org/sqlite"

	"github.com/alecthomas/kong"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/cli"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/flog"
	"github.com/rcy/whatever/version"
)

func main() {
	base, _ := os.UserConfigDir()
	filename := base + "/whatever/flog.sqlite"
	if !version.IsRelease() {
		filename = base + "/whatever-dev/flog.sqlite"
	}
	es, err := flog.NewStore(flog.Config{DBFile: filename})
	if err != nil {
		log.Fatal(err)
	}
	defer es.Close()

	as := app.New(commands.New(es), es)

	kctx := kong.Parse(&cli.CLI)
	err = kctx.Run(as)
	kctx.FatalIfErrorf(err)
}
