package web

import (
	"fmt"

	"github.com/rcy/whatever/commands/service"
)

type ServeCmd struct {
	Port string `default:"9999"`
}

func (c *ServeCmd) Run(s *service.Service) error {
	fmt.Println("serving on port", c.Port)
	return nil
}
