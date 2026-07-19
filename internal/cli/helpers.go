package cli

import "time"

func formatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}
