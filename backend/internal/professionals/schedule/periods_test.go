package schedule

import (
	"errors"
	"reflect"
	"testing"
)

func TestValidatePeriodsSortsAdjacentPeriodsAndAllowsMidnightEnd(t *testing.T) {
	input := []periodInput{
		{Weekday: 7, StartTime: "23:59", EndTime: "24:00"},
		{Weekday: 2, StartTime: "00:00", EndTime: "00:01"},
		{Weekday: 1, StartTime: "12:34", EndTime: "24:00"},
		{Weekday: 1, StartTime: "00:00", EndTime: "12:34"},
	}
	want := []Period{{1, 0, 754}, {1, 754, 1440}, {2, 0, 1}, {7, 1439, 1440}}
	got, err := validatePeriods(input)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("periods = %+v, error = %v, want %+v", got, err, want)
	}
	got, err = validatePeriods([]periodInput{})
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty week = %+v, error = %v", got, err)
	}
}

func TestValidatePeriodsRejectsInvalidWeeks(t *testing.T) {
	for _, test := range []struct {
		name  string
		input []periodInput
	}{
		{"weekday zero", []periodInput{{0, "08:00", "12:00"}}},
		{"weekday eight", []periodInput{{8, "08:00", "12:00"}}},
		{"missing zero", []periodInput{{1, "8:00", "12:00"}}},
		{"seconds", []periodInput{{1, "08:00:00", "12:00"}}},
		{"invalid minute", []periodInput{{1, "08:60", "12:00"}}},
		{"whitespace", []periodInput{{1, "08:00", "12:00 "}}},
		{"crosses midnight", []periodInput{{1, "23:00", "01:00"}}},
		{"midnight start", []periodInput{{1, "24:00", "24:00"}}},
		{"after midnight end", []periodInput{{1, "23:00", "24:01"}}},
		{"same start and end", []periodInput{{1, "12:00", "12:00"}}},
		{"overlap", []periodInput{{1, "10:59", "13:00"}, {1, "08:00", "11:00"}}},
		{"same start", []periodInput{{1, "08:00", "11:00"}, {1, "08:00", "12:00"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got, err := validatePeriods(test.input); !errors.Is(err, ErrInvalidPeriods) || got != nil {
				t.Fatalf("periods = %+v, error = %v", got, err)
			}
		})
	}
}
