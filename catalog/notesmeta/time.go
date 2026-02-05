package notesmeta

import "time"

func Midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func remainingDaysInMonth(now time.Time, nmonths int) int {
	today := Midnight(now)
	y, m, _ := today.Date()

	// First day of nmonths from this month
	firstOfNextMonth := time.Date(y, m+time.Month(nmonths), 1, 0, 0, 0, 0, today.Location())

	return int(firstOfNextMonth.Sub(today).Hours() / 24)
}
