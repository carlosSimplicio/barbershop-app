package onboarding

import "time"

type TrialWindow struct {
	StartsAt time.Time
	EndsAt   time.Time
}

func NewTrialWindow(confirmedAt time.Time) TrialWindow {
	startsAt := confirmedAt.UTC()
	return TrialWindow{StartsAt: startsAt, EndsAt: startsAt.Add(14 * 24 * time.Hour)}
}
