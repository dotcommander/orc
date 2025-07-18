package domain

import (
	"context"
	"time"
)

// Phase represents a processing phase in the domain layer
type Phase interface {
	// Name returns the phase name
	Name() string
	
	// Execute runs the phase with the given input
	Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error)
	
	// ValidateInput validates the input before execution
	ValidateInput(ctx context.Context, input PhaseInput) error
	
	// ValidateOutput validates the output after execution
	ValidateOutput(ctx context.Context, output PhaseOutput) error
	
	// EstimatedDuration returns the estimated time this phase will take
	EstimatedDuration() time.Duration
	
	// CanRetry determines if an error is retryable
	CanRetry(err error) bool
}

// PhaseInput represents input to a phase
type PhaseInput struct {
	// Request is the original user request
	Request string
	
	// Data contains phase-specific input data
	Data interface{}
	
	// Metadata contains additional context
	Metadata map[string]interface{}
}

// PhaseOutput represents output from a phase
type PhaseOutput struct {
	// Data contains the phase output
	Data interface{}
	
	// Error contains any error that occurred
	Error error
	
	// Metadata contains additional context
	Metadata map[string]interface{}
}

// Agent represents an AI agent for domain operations
type Agent interface {
	// Execute sends a prompt to the AI and returns the response
	Execute(ctx context.Context, prompt string, input any) (string, error)
	
	// ExecuteJSON sends a prompt and expects a JSON response
	ExecuteJSON(ctx context.Context, prompt string, input any) (string, error)
}

// Storage represents storage for domain data
type Storage interface {
	// Save stores data with the given key
	Save(ctx context.Context, key string, data []byte) error
	
	// Load retrieves data by key
	Load(ctx context.Context, key string) ([]byte, error)
	
	// Exists checks if a key exists
	Exists(ctx context.Context, key string) bool
	
	// Delete removes data by key
	Delete(ctx context.Context, key string) error
	
	// List returns all keys matching a pattern
	List(ctx context.Context, pattern string) ([]string, error)
}

// CheckpointManager handles checkpointing for domain operations
type CheckpointManager interface {
	// Save creates a checkpoint
	Save(ctx context.Context, sessionID string, phaseIndex int, phaseName string, data interface{}) error
	
	// Load retrieves a checkpoint
	Load(ctx context.Context, sessionID string) (*Checkpoint, error)
	
	// Delete removes a checkpoint
	Delete(ctx context.Context, sessionID string) error
}

// Checkpoint represents a saved state
type Checkpoint struct {
	SessionID  string
	PhaseIndex int
	PhaseName  string
	Data       interface{}
	Timestamp  time.Time
}

// DomainValidator validates domain-specific data
type DomainValidator interface {
	// ValidateRequest validates a user request for this domain
	ValidateRequest(request string) error
	
	// ValidatePhaseTransition validates data between phases
	ValidatePhaseTransition(from, to string, data interface{}) error
}