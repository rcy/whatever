package ids

import (
	"crypto/rand"
	"fmt"
)

func New() string {
	src := make([]byte, 20)
	_, _ = rand.Read(src)
	return fmt.Sprintf("%x", src)
}
