package fiction

import (
	"time"
	
	"github.com/dotcommander/orc/internal/core"
)

type BasePhase struct {
	name              string
	estimatedDuration time.Duration
}

func NewBasePhase(name string, duration time.Duration) BasePhase {
	return BasePhase{
		name:              name,
		estimatedDuration: duration,
	}
}

func (b BasePhase) Name() string {
	return b.name
}

func (b BasePhase) EstimatedDuration() time.Duration {
	return b.estimatedDuration
}

// ValidateInput and ValidateOutput methods removed - 
// Use consolidated validation system from core package instead

func (b BasePhase) CanRetry(err error) bool {
	return core.IsRetryable(err)
}

// Helper functions for logging across fiction phases
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}