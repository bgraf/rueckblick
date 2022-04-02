package dates

import "time"

func FirstDayOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()

	return FromYM(y, int(m))
}

func LastDayOfMonth(t time.Time) time.Time {
	return FirstDayOfMonth(t).AddDate(0, 1, -1)
}

func EqualMonth(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.Month() == t2.Month()
}

func EqualDate(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}

func FromYMD(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

func FromYM(year, month int) time.Time {
	return FromYMD(year, month, 1)
}

func PriorMonday(t time.Time) time.Time {
	if t.Weekday() >= time.Monday {
		return t.AddDate(0, 0, 1-int(t.Weekday()))
	}

	return PriorMonday(t.AddDate(0, 0, -1-int(t.Weekday())))
}

// nextSunday returns the sunday after t or t itself if t is on a sunday.
func NextSunday(t time.Time) time.Time {
	if t.Weekday() == time.Sunday {
		return t
	}

	return t.AddDate(0, 0, 7-int(t.Weekday()))
}

func ForEachDay(start, end time.Time, callback func(time.Time)) {
	for ; !start.After(end); start = start.AddDate(0, 0, 1) {
		callback(start)
	}
}

func AddMonths(t time.Time, offset int) time.Time {
	return t.AddDate(0, offset, 0)
}

func AddYears(t time.Time, offset int) time.Time {
	return t.AddDate(offset, 0, 0)
}
