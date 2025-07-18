package plugin

import (
	"context"
	"fmt"
	
	"github.com/dotcommander/orc/internal/domain"
	domainPlugin "github.com/dotcommander/orc/internal/domain/plugin"
)

// ContextAwarePluginRunner extends the domain plugin runner with context sharing
type ContextAwarePluginRunner struct {
	*domainPlugin.DomainPluginRunner
	contextManager *ContextManager
}

// NewContextAwarePluginRunner creates a plugin runner with context sharing capabilities
func NewContextAwarePluginRunner(
	registry *domainPlugin.DomainRegistry,
	storage domain.Storage,
	contextManager *ContextManager,
) *ContextAwarePluginRunner {
	return &ContextAwarePluginRunner{
		DomainPluginRunner: domainPlugin.NewDomainPluginRunner(registry, storage),
		contextManager:     contextManager,
	}
}

// ExecuteWithContext runs a plugin with context sharing enabled
func (r *ContextAwarePluginRunner) ExecuteWithContext(
	ctx context.Context,
	pluginName string,
	request string,
	sessionID string,
) error {
	// Get the plugin
	registry := r.GetRegistry()
	plugin, err := registry.Get(pluginName)
	if err != nil {
		return err
	}
	
	// Validate request
	if err := plugin.ValidateRequest(request); err != nil {
		return &domainPlugin.DomainInvalidRequestError{
			Plugin: pluginName,
			Reason: err.Error(),
		}
	}
	
	// Create or get context for this session
	var pluginCtx PluginContext
	if existingCtx, exists := r.contextManager.GetContext(sessionID); exists {
		pluginCtx = existingCtx
	} else {
		pluginCtx = r.contextManager.CreateContext(sessionID)
	}
	
	// Initialize shared data
	sharedData := NewSharedData()
	pluginCtx.Set("shared_data", sharedData)
	pluginCtx.Set("plugin_name", pluginName)
	pluginCtx.Set("session_id", sessionID)
	
	// Add plugin context to execution context
	ctxWithPlugin := WithPluginContext(ctx, pluginCtx)
	
	// Get phases and wrap them with context awareness
	phases := plugin.GetPhases()
	wrappedPhases := WrapPhasesWithContext(phases)
	
	// Execute phases sequentially
	input := domain.PhaseInput{
		Request:  request,
		Data:     nil,
		Metadata: make(map[string]interface{}),
	}
	
	for i, phase := range wrappedPhases {
		// Record phase start time
		phaseStart := sharedData.Metrics.StartTime
		
		// Execute phase
		output, err := phase.Execute(ctxWithPlugin, input)
		if err != nil {
			sharedData.AddError(phase.Name(), err, phase.CanRetry(err))
			return &domainPlugin.DomainPhaseExecutionError{
				Plugin:    pluginName,
				Phase:     phase.Name(),
				Err:       err,
				Retryable: phase.CanRetry(err),
			}
		}
		
		// Record phase duration
		phaseDuration := sharedData.Metrics.StartTime.Sub(phaseStart)
		sharedData.RecordPhaseDuration(phase.Name(), phaseDuration)
		
		// Validate output
		if err := phase.ValidateOutput(ctxWithPlugin, output); err != nil {
			return &domainPlugin.DomainPhaseValidationError{
				Plugin: pluginName,
				Phase:  phase.Name(),
				Reason: err.Error(),
			}
		}
		
		// Save intermediate results to storage
		if r.GetStorage() != nil {
			storageKey := fmt.Sprintf("%s/phase_%d_%s.json", sessionID, i+1, phase.Name())
			if dataBytes, err := serializeOutput(output); err == nil {
				r.GetStorage().Save(ctxWithPlugin, storageKey, dataBytes)
			}
		}
		
		// Prepare input for next phase
		if i < len(wrappedPhases)-1 {
			input = domain.PhaseInput{
				Request:  request,
				Data:     output.Data,
				Metadata: output.Metadata,
			}
		}
	}
	
	// Finalize shared data
	sharedData.Finalize()
	
	// Save final context state
	if contextData, err := pluginCtx.MarshalJSON(); err == nil && r.GetStorage() != nil {
		r.GetStorage().Save(ctxWithPlugin, fmt.Sprintf("%s/context.json", sessionID), contextData)
	}
	
	return nil
}

// GetRegistry returns the underlying domain registry
func (r *ContextAwarePluginRunner) GetRegistry() *domainPlugin.DomainRegistry {
	// This would need to be exposed in the actual implementation
	// For now, returning nil as a placeholder
	return nil
}

// GetStorage returns the underlying storage
func (r *ContextAwarePluginRunner) GetStorage() domain.Storage {
	// This would need to be exposed in the actual implementation
	// For now, returning nil as a placeholder
	return nil
}

// serializeOutput converts phase output to JSON bytes
func serializeOutput(output domain.PhaseOutput) ([]byte, error) {
	// Simple JSON serialization
	// In production, you might want more sophisticated serialization
	return []byte(fmt.Sprintf(`{"data": %v, "metadata": %v}`, output.Data, output.Metadata)), nil
}

// Example of how to use the context-aware runner in main.go:
/*
func main() {
    // Create dependencies
    storage := createStorage()
    registry := domainPlugin.NewDomainRegistry()
    
    // Register plugins
    registry.Register(fictionPlugin)
    registry.Register(codePlugin)
    
    // Create context manager
    contextManager := plugin.NewContextManager(
        plugin.WithTTL(24 * time.Hour),
        plugin.WithCleanupInterval(1 * time.Hour),
    )
    defer contextManager.Stop()
    
    // Create context-aware runner
    runner := plugin.NewContextAwarePluginRunner(registry, storage, contextManager)
    
    // Execute with context
    ctx := context.Background()
    sessionID := generateSessionID()
    
    err := runner.ExecuteWithContext(ctx, "code", "Build a REST API", sessionID)
    if err != nil {
        log.Fatal(err)
    }
}
*/