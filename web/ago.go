package web

import (
	"time"

	"github.com/hako/durafmt"
)

func ago(ts time.Time) string {
	return durafmt.Parse(time.Since(ts)).LimitFirstN(1).String() + " ago"
}
