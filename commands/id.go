package commands

import (
	"crypto/rand"
	"fmt"
)

func makeID() string {
	src := make([]byte, 20)
	_, _ = rand.Read(src)
	return fmt.Sprintf("%x", src)
}

type IDCmd struct {
	Aggregate string `arg:"" default:"note"`
}

func (c *IDCmd) Run(ctx *Context) error {
	fmt.Println(makeID())
	return nil
}
