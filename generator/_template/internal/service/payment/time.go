package payment

import "time"

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}
