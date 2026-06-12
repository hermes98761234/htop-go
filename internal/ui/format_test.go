package ui

import "testing"

func TestFormatMeter(t *testing.T) {
	cases := []struct {
		kb   uint64
		want string
	}{
		{800, "800K"},
		{512000, "500M"},
		{3355443, "3.20G"},
		{15728640, "15.0G"},
	}
	for _, c := range cases {
		if got := FormatMeter(c.kb); got != c.want {
			t.Errorf("FormatMeter(%d) = %q, want %q", c.kb, got, c.want)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		sec  float64
		want string
	}{
		{3784, "01:03:04"},
		{93784, "1 day, 02:03:04"},
		{200000, "2 days, 07:33:20"},
	}
	for _, c := range cases {
		if got := FormatUptime(c.sec); got != c.want {
			t.Errorf("FormatUptime(%f) = %q, want %q", c.sec, got, c.want)
		}
	}
}
