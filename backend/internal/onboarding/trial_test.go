package onboarding

import (
	"testing"
	"time"
)

func TestNewTrialWindowStartsAtConfirmationAndLastsExactly14Days(t *testing.T) {
	confirmedAt := time.Date(2026, time.October, 18, 16, 30, 0, 0, time.FixedZone("test", -3*60*60))

	window := NewTrialWindow(confirmedAt)
	wantStart := confirmedAt.UTC()
	wantEnd := wantStart.Add(14 * 24 * time.Hour)
	if !window.StartsAt.Equal(wantStart) || window.StartsAt.Location() != time.UTC {
		t.Fatalf("StartsAt = %s, want UTC %s", window.StartsAt, wantStart)
	}
	if !window.EndsAt.Equal(wantEnd) || window.EndsAt.Location() != time.UTC {
		t.Fatalf("EndsAt = %s, want UTC %s", window.EndsAt, wantEnd)
	}
}
