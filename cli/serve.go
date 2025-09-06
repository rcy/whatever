package cli

import (
	"fmt"
	"net/http"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/web"
)

type ServeCmd struct {
	Port string `default:"9999" env:"PORT"`
}

func (c *ServeCmd) Run(app *app.Service) error {
	ws := web.Server(app)
	fmt.Printf("listening on http://localhost:%s\n", c.Port)
	return http.ListenAndServe(":"+c.Port, ws)
}
