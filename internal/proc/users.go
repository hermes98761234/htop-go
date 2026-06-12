package proc

import (
	"os"
	"strconv"
	"strings"
)

// UserTable maps numeric UIDs to user names.
type UserTable struct {
	names map[int]string
}

// ParsePasswd builds a UserTable from /etc/passwd content.
func ParsePasswd(data string) *UserTable {
	names := make(map[int]string)
	for _, line := range strings.Split(data, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		names[uid] = parts[0]
	}
	return &UserTable{names: names}
}

// LoadUsers reads /etc/passwd; on error returns an empty table (Name then
// falls back to numeric UIDs).
func LoadUsers() *UserTable {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return &UserTable{names: map[int]string{}}
	}
	return ParsePasswd(string(data))
}

// Name returns the user name for uid, or the uid as a string if unknown.
func (u *UserTable) Name(uid int) string {
	if name, ok := u.names[uid]; ok {
		return name
	}
	return strconv.Itoa(uid)
}
