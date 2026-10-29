package onboarding

import "time"

const (
	TrialDurationDays = 14
	TrialDuration     = TrialDurationDays * 24 * time.Hour
)

type TrialWindow struct {
	StartsAt time.Time
	EndsAt   time.Time
}

func NewTrialWindow(confirmedAt time.Time) TrialWindow {
	startsAt := confirmedAt.UTC()
	return TrialWindow{StartsAt: startsAt, EndsAt: startsAt.Add(TrialDuration)}
}
