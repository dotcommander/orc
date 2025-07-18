package plugin

import "fmt"

// DomainPluginAlreadyRegisteredError indicates a plugin is already registered
type DomainPluginAlreadyRegisteredError struct {
	Name string
}

func (e *DomainPluginAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("plugin '%s' is already registered", e.Name)
}

// DomainPluginNotFoundError indicates a plugin was not found
type DomainPluginNotFoundError struct {
	Name string
}

func (e *DomainPluginNotFoundError) Error() string {
	return fmt.Sprintf("plugin '%s' not found", e.Name)
}

// DomainInvalidRequestError indicates an invalid request for a plugin
type DomainInvalidRequestError struct {
	Plugin string
	Reason string
}

func (e *DomainInvalidRequestError) Error() string {
	return fmt.Sprintf("invalid request for plugin '%s': %s", e.Plugin, e.Reason)
}

// DomainPhaseExecutionError indicates a phase execution failure
type DomainPhaseExecutionError struct {
	Phase   string
	Plugin  string
	Err     error
	Retryable bool
}

func (e *DomainPhaseExecutionError) Error() string {
	return fmt.Sprintf("phase '%s' failed for plugin '%s': %v", e.Phase, e.Plugin, e.Err)
}

func (e *DomainPhaseExecutionError) Unwrap() error {
	return e.Err
}

func (e *DomainPhaseExecutionError) IsRetryable() bool {
	return e.Retryable
}

// DomainValidationError indicates domain-specific validation failure
type DomainValidationError struct {
	Type   string
	Field  string
	Value  interface{}
	Reason string
}

func (e *DomainValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s.%s (value: %v): %s", e.Type, e.Field, e.Value, e.Reason)
}

// DomainTransformationError indicates data transformation failure
type DomainTransformationError struct {
	FromType string
	ToType   string
	Err      error
}

func (e *DomainTransformationError) Error() string {
	return fmt.Sprintf("transformation from %s to %s failed: %v", e.FromType, e.ToType, e.Err)
}

func (e *DomainTransformationError) Unwrap() error {
	return e.Err
}

// DomainPhaseValidationError indicates a phase validation failure
type DomainPhaseValidationError struct {
	Plugin string
	Phase  string
	Reason string
}

func (e *DomainPhaseValidationError) Error() string {
	return fmt.Sprintf("validation failed for phase '%s' in plugin '%s': %s", e.Phase, e.Plugin, e.Reason)
}