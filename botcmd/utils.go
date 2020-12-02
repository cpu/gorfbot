package botcmd

import "time"

// FormatTime formats a given time.Time instance as UTC in a simple format.
func FormatTime(d time.Time) string {
	layout := "Mon Jan 2 2006 15:04:05 UTC"
	return d.Format(layout)
}
