package helpers

import "time"

func FormatDate(d string) string {
	date, err := time.Parse("2006-01-02T15:04:05", d)
	if err != nil {
		return "Invalid date"
	}
	return date.Format("Jan 2, 2006 (Mon)")
}

func FormatTime(t string) string {
	_time, err := time.Parse("2006-01-02T15:04:05", t)
	if err != nil {
		return "Invalid time"
	}
	return _time.Format("3:04pm")
}
