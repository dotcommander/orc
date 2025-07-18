package code

import (
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// BasePhase provides common phase functionality
type BasePhase struct {
	name              string
	estimatedDuration time.Duration
}

// NewBasePhase creates a new base phase
func NewBasePhase(name string, duration time.Duration) BasePhase {
	return BasePhase{
		name:              name,
		estimatedDuration: duration,
	}
}

// Name returns the phase name
func (b BasePhase) Name() string {
	return b.name
}

// EstimatedDuration returns expected phase duration
func (b BasePhase) EstimatedDuration() time.Duration {
	return b.estimatedDuration
}

// CanRetry determines if an error is retryable
func (b BasePhase) CanRetry(err error) bool {
	return core.IsRetryable(err)
}

// ValidateInput and ValidateOutput methods removed - 
// Use consolidated validation system from core package instead