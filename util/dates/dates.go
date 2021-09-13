package dates

import "time"

func FirstDayOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()

	return time.Date(y, m, 1, 0, 0, 0, 0, time.Local)
}
