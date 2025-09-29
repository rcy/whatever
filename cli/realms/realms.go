package realms

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/realm"
)

type Cmd struct {
	List   listCmd   `cmd:"" default:"withargs" aliases:"ls"`
	Add    addCmd    `cmd:""`
	Delete deleteCmd `cmd:""`
}

type listCmd struct {
	Deleted bool `help:"Show deleted realms"`
}

func (c *listCmd) Run(app *app.App) error {
	var realmList []realm.Realm
	var err error
	realmList, err = app.Realms.FindAll()
	if err != nil {
		return err
	}
	for _, realm := range realmList {
		fmt.Printf("%s %s\n", realm.ID, realm.Name)
	}

	return nil
}

type addCmd struct {
	Text []string `arg:""`
}

func (c *addCmd) Run(app *app.App) error {
	realmID := uuid.New()
	err := app.Commander.Send(commands.CreateRealm{RealmID: realmID, Name: strings.Join(c.Text, " ")})
	fmt.Println(realmID)
	return err
}

type deleteCmd struct {
	ID uuid.UUID `arg:""`
}

func (c *deleteCmd) Run(app *app.App) error {
	return app.Commander.Send(commands.DeleteRealm{RealmID: c.ID})
}
