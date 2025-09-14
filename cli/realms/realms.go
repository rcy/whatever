package realms

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/projections/realms"
)

type Cmd struct {
	List listCmd `cmd:"" default:"withargs" aliases:"ls"`
	Add  addCmd  `cmd:""`
}

type listCmd struct {
	Deleted bool `help:"Show deleted realms"`
}

func (c *listCmd) Run(app *app.App) error {
	var realmList []realms.Realm
	var err error
	realmList, err = app.Realms.FindAll()
	if err != nil {
		return err
	}
	for _, realm := range realmList {
		fmt.Printf("%s %s\n", realm.ID[0:7], realm.Name)
	}

	return nil
}

type addCmd struct {
	Text []string `arg:""`
}

func (c *addCmd) Run(app *app.App) error {
	aggID, err := app.Commands.CreateRealm(strings.Join(c.Text, " "))
	fmt.Println(aggID)
	return err
}
