package cli

import (
	"github.com/pkg/browser"
	"github.com/rcy/whatever/app"
)

type BugCmd struct {
}

func (c *BugCmd) Run(app *app.App) error {
	return browser.OpenURL("https://github.com/rcy/whatever/issues/new")
}
