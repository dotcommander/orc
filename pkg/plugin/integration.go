package plugin

import (
	"context"
	"fmt"
	
	"github.com/vampirenirmal/orchestrator/internal/domain"
)

// ContextAwarePhase wraps a domain.Phase to provide context sharing capabilities
type ContextAwarePhase struct {
	domain.Phase
	contextKey string
}

// NewContextAwarePhase creates a new context-aware phase wrapper
func NewContextAwarePhase(phase domain.Phase) *ContextAwarePhase {
	return &ContextAwarePhase{
		Phase:      phase,
		contextKey: fmt.Sprintf("phase_%s", phase.Name()),
	}
}

// Execute runs the phase with context sharing
func (cap *ContextAwarePhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Get the plugin context
	pluginCtx, err := GetPluginContext(ctx)
	if err != nil {
		// If no context exists, execute normally
		return cap.Phase.Execute(ctx, input)
	}
	
	// Store phase input in context
	pluginCtx.Set(fmt.Sprintf("%s_input", cap.contextKey), input)
	
	// Check if we have shared data
	var sharedData *SharedData
	if sd, exists := pluginCtx.Get("shared_data"); exists {
		if data, ok := sd.(*SharedData); ok {
			sharedData = data
		}
	} else {
		// Create shared data if it doesn't exist
		sharedData = NewSharedData()
		pluginCtx.Set("shared_data", sharedData)
	}
	
	// Add previous phase outputs to input metadata
	if input.Metadata == nil {
		input.Metadata = make(map[string]interface{})
	}
	
	// Make all previous phase outputs available
	for phaseName, output := range sharedData.PhaseOutputs {
		input.Metadata[fmt.Sprintf("phase_%s_output", phaseName)] = output
	}
	
	// Execute the actual phase
	output, err := cap.Phase.Execute(ctx, input)
	
	// Store the output in shared data
	if err == nil {
		sharedData.SetPhaseOutput(cap.Phase.Name(), output.Data)
		pluginCtx.Set(fmt.Sprintf("%s_output", cap.contextKey), output)
	} else {
		// Record the error
		sharedData.AddError(cap.Phase.Name(), err, cap.Phase.CanRetry(err))
	}
	
	return output, err
}

// WrapPhasesWithContext wraps all phases to be context-aware
func WrapPhasesWithContext(phases []domain.Phase) []domain.Phase {
	wrapped := make([]domain.Phase, len(phases))
	for i, phase := range phases {
		wrapped[i] = NewContextAwarePhase(phase)
	}
	return wrapped
}

// ExecuteWithContext runs a plugin with context sharing enabled
func ExecuteWithContext(
	ctx context.Context,
	runner interface {
		Execute(ctx context.Context, pluginName, request string) error
	},
	pluginName string,
	request string,
	contextManager *ContextManager,
	sessionID string,
) error {
	// Create or get context for this session
	var pluginCtx PluginContext
	if existingCtx, exists := contextManager.GetContext(sessionID); exists {
		pluginCtx = existingCtx
	} else {
		pluginCtx = contextManager.CreateContext(sessionID)
	}
	
	// Add the plugin context to the execution context
	ctxWithPlugin := WithPluginContext(ctx, pluginCtx)
	
	// Execute the plugin
	return runner.Execute(ctxWithPlugin, pluginName, request)
}

// PhaseContextHelper provides convenience methods for phases to access shared context
type PhaseContextHelper struct {
	ctx context.Context
	pluginCtx PluginContext
}

// NewPhaseContextHelper creates a new helper for the given context
func NewPhaseContextHelper(ctx context.Context) (*PhaseContextHelper, error) {
	pluginCtx, err := GetPluginContext(ctx)
	if err != nil {
		return nil, err
	}
	
	return &PhaseContextHelper{
		ctx:       ctx,
		pluginCtx: pluginCtx,
	}, nil
}

// GetPreviousPhaseOutput retrieves output from a previous phase
func (pch *PhaseContextHelper) GetPreviousPhaseOutput(phaseName string) (interface{}, error) {
	sd, err := pch.getSharedData()
	if err != nil {
		return nil, err
	}
	
	output, exists := sd.GetPhaseOutput(phaseName)
	if !exists {
		return nil, fmt.Errorf("output for phase %s not found", phaseName)
	}
	
	return output, nil
}

// SetMetadata stores metadata that will be available to all subsequent phases
func (pch *PhaseContextHelper) SetMetadata(key string, value interface{}) error {
	sd, err := pch.getSharedData()
	if err != nil {
		return err
	}
	
	sd.Metadata[key] = value
	return nil
}

// GetMetadata retrieves metadata set by previous phases
func (pch *PhaseContextHelper) GetMetadata(key string) (interface{}, error) {
	sd, err := pch.getSharedData()
	if err != nil {
		return nil, err
	}
	
	value, exists := sd.Metadata[key]
	if !exists {
		return nil, fmt.Errorf("metadata key %s not found", key)
	}
	
	return value, nil
}

// getSharedData retrieves the shared data from the plugin context
func (pch *PhaseContextHelper) getSharedData() (*SharedData, error) {
	value, exists := pch.pluginCtx.Get("shared_data")
	if !exists {
		return nil, fmt.Errorf("shared data not found in context")
	}
	
	sd, ok := value.(*SharedData)
	if !ok {
		return nil, fmt.Errorf("shared data has invalid type")
	}
	
	return sd, nil
}