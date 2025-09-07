package flog

import (
	"crypto/rand"
	"fmt"
)

func ID() string {
	src := make([]byte, 20)
	_, _ = rand.Read(src)
	return fmt.Sprintf("%x", src)
}
