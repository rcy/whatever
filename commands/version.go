package commands

import (
	"fmt"

	"github.com/rcy/whatever/version"
)

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Printf("version=%s isRelease=%v", version.Version(), version.IsRelease())
	return nil
}
