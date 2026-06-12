package proc

import "testing"

const pidStatFixture = `1234 (tmux: server) S 1 1234 1234 0 -1 4194304 1000 0 0 0 250 150 0 0 20 0 3 0 100000 123456789 4321 18446744073709551615 1 1 0 0 0 0 0 3670020 1216 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0`

func TestParsePIDStat(t *testing.T) {
	f, err := ParsePIDStat(pidStatFixture)
	if err != nil {
		t.Fatalf("ParsePIDStat error: %v", err)
	}
	if f.PID != 1234 {
		t.Errorf("PID = %d, want 1234", f.PID)
	}
	if f.Comm != "tmux: server" {
		t.Errorf("Comm = %q, want \"tmux: server\"", f.Comm)
	}
	if f.State != 'S' {
		t.Errorf("State = %c, want S", f.State)
	}
	if f.PPID != 1 {
		t.Errorf("PPID = %d, want 1", f.PPID)
	}
	if f.UTime != 250 || f.STime != 150 {
		t.Errorf("UTime,STime = %d,%d want 250,150", f.UTime, f.STime)
	}
	if f.Priority != 20 || f.Nice != 0 {
		t.Errorf("Priority,Nice = %d,%d want 20,0", f.Priority, f.Nice)
	}
	if f.Threads != 3 {
		t.Errorf("Threads = %d, want 3", f.Threads)
	}
	if f.StartTime != 100000 {
		t.Errorf("StartTime = %d, want 100000", f.StartTime)
	}
	if f.VSizeBytes != 123456789 {
		t.Errorf("VSizeBytes = %d, want 123456789", f.VSizeBytes)
	}
	if f.RSSPages != 4321 {
		t.Errorf("RSSPages = %d, want 4321", f.RSSPages)
	}
}

func TestParseStatmShared(t *testing.T) {
	shared, err := ParseStatmShared("12345 4321 999 100 0 5000 0\n")
	if err != nil {
		t.Fatalf("ParseStatmShared error: %v", err)
	}
	if shared != 999 {
		t.Errorf("shared = %d, want 999", shared)
	}
}

const statusFixture = `Name:	bash
Umask:	0022
State:	S (sleeping)
Tgid:	1234
Pid:	1234
PPid:	1
Uid:	1000	1000	1000	1000
Gid:	1000	1000	1000	1000
`

func TestParseStatusUID(t *testing.T) {
	uid, err := ParseStatusUID(statusFixture)
	if err != nil {
		t.Fatalf("ParseStatusUID error: %v", err)
	}
	if uid != 1000 {
		t.Errorf("uid = %d, want 1000", uid)
	}
}

func TestParseCmdline(t *testing.T) {
	got := ParseCmdline([]byte("/usr/bin/foo\x00--bar\x00baz\x00"), "foo")
	if got != "/usr/bin/foo --bar baz" {
		t.Errorf("got %q", got)
	}
	// Kernel threads have an empty cmdline: fall back to [comm].
	got = ParseCmdline(nil, "kthreadd")
	if got != "[kthreadd]" {
		t.Errorf("got %q, want [kthreadd]", got)
	}
}

const passwdFixture = `root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
user:x:1000:1000::/home/user:/bin/bash
`

func TestUserTable(t *testing.T) {
	u := ParsePasswd(passwdFixture)
	if got := u.Name(0); got != "root" {
		t.Errorf("Name(0) = %q, want root", got)
	}
	if got := u.Name(1000); got != "user" {
		t.Errorf("Name(1000) = %q, want user", got)
	}
	// Unknown uid falls back to the number itself.
	if got := u.Name(4242); got != "4242" {
		t.Errorf("Name(4242) = %q, want 4242", got)
	}
}
