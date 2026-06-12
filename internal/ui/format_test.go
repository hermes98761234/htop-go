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

func TestFormatSize(t *testing.T) {
	cases := []struct {
		kb   uint64
		want string
	}{
		{0, "0"},
		{99999, "99999"},
		{100000, "97M"},
		{2097152, "2048M"},
		{20971520, "20.0G"},
	}
	for _, c := range cases {
		if got := FormatSize(c.kb); got != c.want {
			t.Errorf("FormatSize(%d) = %q, want %q", c.kb, got, c.want)
		}
	}
}

func TestFormatTimePlus(t *testing.T) {
	if got := FormatTimePlus(12345, 100); got != "2:03.45" {
		t.Errorf("got %q, want 2:03.45", got)
	}
	if got := FormatTimePlus(372300, 100); got != "1:02:03" {
		t.Errorf("got %q, want 1:02:03", got)
	}
}
