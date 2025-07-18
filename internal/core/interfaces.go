package core

import (
	"context"
	"time"
)

type Phase interface {
	Name() string
	Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error)
	ValidateInput(ctx context.Context, input PhaseInput) error
	ValidateOutput(ctx context.Context, output PhaseOutput) error
	EstimatedDuration() time.Duration
	CanRetry(err error) bool
}

type PhaseInput struct {
	Request   string
	Prompt    string
	Data      interface{}
	SessionID string                 // Added for resume functionality
	Metadata  map[string]interface{} // Additional context
}

type PhaseOutput struct {
	Data     interface{}
	Error    error
	Metadata map[string]interface{} // Additional context
}

type Agent interface {
	Execute(ctx context.Context, prompt string, input any) (string, error)
	ExecuteJSON(ctx context.Context, prompt string, input any) (string, error)
}

type Storage interface {
	Save(ctx context.Context, path string, data []byte) error
	Load(ctx context.Context, path string) ([]byte, error)
	List(ctx context.Context, pattern string) ([]string, error)
	Exists(ctx context.Context, path string) bool
	Delete(ctx context.Context, path string) error
}

type DomainValidator interface {
	ValidateInput(input interface{}) error
	ValidateOutput(output interface{}) error
}

type DomainTransformer interface {
	Transform(ctx context.Context, input interface{}) (interface{}, error)
	GetInputType() string
	GetOutputType() string
}