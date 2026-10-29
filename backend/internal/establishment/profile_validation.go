package establishment

import (
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"
)

func IsNameValid(name string) bool {
	return utf8.ValidString(name) && strings.TrimSpace(name) != "" && utf8.RuneCountInString(name) <= MaxNameCodePoints
}

func IsTimeZoneValid(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}
