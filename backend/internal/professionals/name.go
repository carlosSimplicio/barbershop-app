package professionals

import (
	"strings"
	"unicode/utf8"
)

func IsNameValid(name string) bool {
	return utf8.ValidString(name) && strings.TrimSpace(name) != "" && utf8.RuneCountInString(name) <= MaxNameCodePoints && !strings.ContainsRune(name, '\x00')
}
