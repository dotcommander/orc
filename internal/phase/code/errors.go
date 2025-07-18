package code

import (
	"github.com/dotcommander/orc/internal/core"
)

// Re-export centralized error types for backward compatibility
type PhaseError = core.PhaseError
type ValidationError = core.ValidationError
type GenerationError = core.GenerationError
type RecoveryManager = core.RecoveryManager

// Re-export constructor functions
var (
	NewRecoveryManager = core.NewRecoveryManager
	NewPhaseError      = core.NewPhaseError
	NewValidationError = core.NewValidationError
	NewGenerationError = core.NewGenerationError
)