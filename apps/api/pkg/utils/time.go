package utils

import "time"

// A zero time.Time formats to "0001-01-01T00:00:00Z", which every JSON client
// happily parses as a real date. Absent has to look absent.
func RFC3339OrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
