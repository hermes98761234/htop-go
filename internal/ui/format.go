package ui

import "fmt"

// FormatMeter renders a kB amount for meter texts: "800K", "500M", "3.20G".
func FormatMeter(kb uint64) string {
	g := float64(kb) / (1024 * 1024)
	switch {
	case g >= 10:
		return fmt.Sprintf("%.1fG", g)
	case g >= 1:
		return fmt.Sprintf("%.2fG", g)
	case kb >= 1024:
		return fmt.Sprintf("%dM", kb/1024)
	default:
		return fmt.Sprintf("%dK", kb)
	}
}

// FormatUptime renders seconds as "hh:mm:ss" with an optional day prefix.
func FormatUptime(sec float64) string {
	t := int64(sec)
	days := t / 86400
	h := t % 86400 / 3600
	m := t % 3600 / 60
	s := t % 60
	switch {
	case days == 1:
		return fmt.Sprintf("1 day, %02d:%02d:%02d", h, m, s)
	case days > 1:
		return fmt.Sprintf("%d days, %02d:%02d:%02d", days, h, m, s)
	default:
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
}
