// Package pagination defines the limits shared by the authenticated list APIs.
package pagination

import "math"

const (
	MinLimit      = 1
	DefaultLimit  = 20
	MaxLimit      = 100
	MinOffset     = 0
	DefaultOffset = MinOffset
	MaxOffset     = math.MaxInt32
)
