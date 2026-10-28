package auth

import "time"

// SessionLifetime governs sessions, access tokens and their cookies without renewal.
const SessionLifetime = 15 * 24 * time.Hour
