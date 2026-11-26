package schedule

import (
	"cmp"
	"errors"
	"slices"
)

const (
	minutesPerHour = 60
	hoursPerDay    = 24
	minutesPerDay  = hoursPerDay * minutesPerHour
	daysPerWeek    = 7
	maxPeriods     = daysPerWeek * minutesPerDay
)

var ErrInvalidPeriods = errors.New("invalid professional work periods")

type periodInput struct {
	Weekday   int
	StartTime string
	EndTime   string
}

func validatePeriods(input []periodInput) ([]Period, error) {
	if input == nil || len(input) > maxPeriods {
		return nil, ErrInvalidPeriods
	}
	periods := make([]Period, 0, len(input))
	for _, period := range input {
		start, validStart := parseLocalTime(period.StartTime)
		end, validEnd := parseLocalTime(period.EndTime)
		if period.Weekday < 1 || period.Weekday > daysPerWeek || !validStart || !validEnd || start >= end {
			return nil, ErrInvalidPeriods
		}
		periods = append(periods, Period{Weekday: int16(period.Weekday), StartMinute: start, EndMinute: end})
	}
	slices.SortFunc(periods, func(left, right Period) int {
		if left.Weekday != right.Weekday {
			return cmp.Compare(left.Weekday, right.Weekday)
		}
		return cmp.Compare(left.StartMinute, right.StartMinute)
	})
	for index := 1; index < len(periods); index++ {
		previous, current := periods[index-1], periods[index]
		if current.Weekday == previous.Weekday && current.StartMinute < previous.EndMinute {
			return nil, ErrInvalidPeriods
		}
	}
	return periods, nil
}

func parseLocalTime(value string) (int16, bool) {
	if len(value) != len("00:00") || value[2] != ':' {
		return 0, false
	}
	for _, digit := range []byte{value[0], value[1], value[3], value[4]} {
		if digit < '0' || digit > '9' {
			return 0, false
		}
	}
	hour := int16(value[0]-'0')*10 + int16(value[1]-'0')
	minute := int16(value[3]-'0')*10 + int16(value[4]-'0')
	if minute >= minutesPerHour || hour > hoursPerDay || (hour == hoursPerDay && minute != 0) {
		return 0, false
	}
	return hour*minutesPerHour + minute, true
}
