// Package orc provides the public API for Orc plugins
package orc

import (
	"context"
	"time"
)

// Agent represents an AI agent that plugins can use
type Agent interface {
	// Execute sends a prompt to the AI and returns the response
	Execute(ctx context.Context, prompt string, input interface{}) (string, error)
	
	// ExecuteJSON sends a prompt and expects a JSON response
	ExecuteJSON(ctx context.Context, prompt string, input interface{}) (string, error)
}

// Storage provides persistent storage for plugins
type Storage interface {
	// Save stores data with a key
	Save(ctx context.Context, key string, data []byte) error
	
	// Load retrieves data by key
	Load(ctx context.Context, key string) ([]byte, error)
	
	// Exists checks if a key exists
	Exists(ctx context.Context, key string) bool
	
	// Delete removes data by key
	Delete(ctx context.Context, key string) error
	
	// List returns keys matching a pattern
	List(ctx context.Context, pattern string) ([]string, error)
	
	// SaveOutput saves output to the session's output directory
	SaveOutput(sessionID, filename string, data []byte) error
}

// Phase represents a step in the content generation pipeline
type Phase interface {
	// Name returns the phase name
	Name() string
	
	// Execute runs the phase
	Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error)
	
	// ValidateInput checks if the input is valid
	ValidateInput(ctx context.Context, input PhaseInput) error
	
	// ValidateOutput checks if the output is valid
	ValidateOutput(ctx context.Context, output PhaseOutput) error
	
	// EstimatedDuration returns how long this phase typically takes
	EstimatedDuration() time.Duration
	
	// CanRetry indicates if this phase can be retried on failure
	CanRetry(err error) bool
}

// PhaseInput contains data passed to a phase
type PhaseInput struct {
	// Request is the original user request
	Request string
	
	// Data contains phase-specific input data
	Data interface{}
	
	// Metadata contains additional context
	Metadata map[string]interface{}
	
	// SessionID identifies the current session
	SessionID string
	
	// PreviousOutputs contains outputs from earlier phases
	PreviousOutputs map[string]interface{}
}

// PhaseOutput contains results from a phase
type PhaseOutput struct {
	// Data contains the phase output
	Data interface{}
	
	// Error indicates if an error occurred
	Error error
	
	// Metadata contains additional information
	Metadata map[string]interface{}
}

// Plugin defines the interface that all Orc plugins must implement
type Plugin interface {
	// GetInfo returns plugin metadata
	GetInfo() PluginInfo
	
	// CreatePhases returns the phases this plugin provides
	CreatePhases() ([]Phase, error)
	
	// ValidateRequest checks if a request is suitable for this plugin
	ValidateRequest(request string) error
	
	// GetOutputSpec describes what outputs this plugin produces
	GetOutputSpec() OutputSpec
	
	// GetPhaseTimeouts returns recommended timeouts for each phase
	GetPhaseTimeouts() map[string]time.Duration
	
	// GetRequiredConfig returns required configuration keys
	GetRequiredConfig() []string
	
	// GetDefaultConfig returns default configuration values
	GetDefaultConfig() map[string]interface{}
}

// PluginInfo contains plugin metadata
type PluginInfo struct {
	// Name is the plugin identifier
	Name string
	
	// Version is the plugin version
	Version string
	
	// Description explains what the plugin does
	Description string
	
	// Author is the plugin creator
	Author string
	
	// Domains lists the content domains this plugin handles
	Domains []string
	
	// MinOrcVersion is the minimum Orc version required
	MinOrcVersion string
}

// OutputSpec describes plugin outputs
type OutputSpec struct {
	// PrimaryOutput is the main output file/directory
	PrimaryOutput string
	
	// SecondaryOutputs lists additional outputs
	SecondaryOutputs []string
	
	// FilePatterns maps output types to glob patterns
	FilePatterns map[string]string
}

// AgentFactory creates AI agents for plugins
type AgentFactory interface {
	// CreateAgent creates an agent with a specific role and prompt file
	CreateAgent(role, promptPath string) Agent
}

// Logger provides structured logging for plugins
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	With(key string, value interface{}) Logger
}