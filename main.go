package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/cli"
	"github.com/rcy/whatever/version"
)

func main() {
	// filename, err := getFilename()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	a := app.New()

	// noteID := uuid.New()
	// realmID := uuid.New()
	// cmds := []evoke.Command{
	// 	//commands.CreateRealmCommand{RealmID: realmID, Name: "MyRealm"},
	// 	commands.CreateNoteCommand{NoteID: noteID, Text: "hello", RealmID: realmID},
	// 	// commands.CreateNoteCommand{NoteID: uuid.New(), Text: "random", RealmID: realmID},
	// 	//commands.DeleteNoteCommand{NoteID: noteID},
	// 	// commands.UndeleteNoteCommand{NoteID: noteID},
	// }
	// for _, cmd := range cmds {
	// 	err := a.CommandBus().Send(cmd)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	//a.CommandBus().MustSend(commands.CreateRealmCommand{RealmID: uuid.New(), Name: "MyRealm"})

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
