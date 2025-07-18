package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/domain"
)

// DomainPlugin represents a domain-specific plugin
type DomainPlugin interface {
	// Name returns the plugin name (e.g., "fiction", "code", "docs")
	Name() string
	
	// Description returns a human-readable description
	Description() string
	
	// GetPhases returns the ordered phases for this task type
	GetPhases() []domain.Phase
	
	// GetDefaultConfig returns default configuration for this plugin
	GetDefaultConfig() DomainPluginConfig
	
	// ValidateRequest validates if the user request is appropriate for this plugin
	ValidateRequest(request string) error
	
	// GetOutputSpec returns the expected output structure
	GetOutputSpec() DomainOutputSpec
	
	// GetDomainValidator returns domain-specific validation
	GetDomainValidator() domain.DomainValidator
}

// DomainPluginConfig holds plugin-specific configuration
type DomainPluginConfig struct {
	// Prompts maps phase names to prompt file paths
	Prompts map[string]string `yaml:"prompts"`
	
	// Limits defines plugin-specific resource limits
	Limits DomainPluginLimits `yaml:"limits"`
	
	// OutputDir override for this plugin (optional)
	OutputDir string `yaml:"output_dir,omitempty"`
	
	// Metadata for plugin behavior
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

// DomainPluginLimits defines resource constraints for a plugin
type DomainPluginLimits struct {
	// MaxConcurrentPhases limits parallel phase execution
	MaxConcurrentPhases int `yaml:"max_concurrent_phases"`
	
	// PhaseTimeouts maps phase names to timeout durations
	PhaseTimeouts map[string]time.Duration `yaml:"phase_timeouts"`
	
	// MaxRetries for failed phases
	MaxRetries int `yaml:"max_retries"`
	
	// TotalTimeout for entire plugin execution
	TotalTimeout time.Duration `yaml:"total_timeout"`
}

// DomainOutputSpec describes the expected outputs from a plugin
type DomainOutputSpec struct {
	// PrimaryOutput is the main file users care about
	PrimaryOutput string `yaml:"primary_output"`
	
	// SecondaryOutputs are supporting files
	SecondaryOutputs []string `yaml:"secondary_outputs"`
	
	// Description maps output files to user-friendly descriptions
	Descriptions map[string]string `yaml:"descriptions"`
}

// DomainRegistry manages domain plugin registration and discovery
type DomainRegistry struct {
	plugins map[string]DomainPlugin
}

// NewDomainRegistry creates a new domain plugin registry
func NewDomainRegistry() *DomainRegistry {
	return &DomainRegistry{
		plugins: make(map[string]DomainPlugin),
	}
}

// Register adds a plugin to the registry
func (r *DomainRegistry) Register(plugin DomainPlugin) error {
	if r.plugins[plugin.Name()] != nil {
		return &DomainPluginAlreadyRegisteredError{Name: plugin.Name()}
	}
	r.plugins[plugin.Name()] = plugin
	return nil
}

// Replace replaces an existing plugin in the registry or adds it if not present
func (r *DomainRegistry) Replace(plugin DomainPlugin) {
	r.plugins[plugin.Name()] = plugin
}

// Get retrieves a plugin by name
func (r *DomainRegistry) Get(name string) (DomainPlugin, error) {
	plugin, exists := r.plugins[name]
	if !exists {
		return nil, &DomainPluginNotFoundError{Name: name}
	}
	return plugin, nil
}

// List returns all registered plugin names
func (r *DomainRegistry) List() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// GetPlugins returns all registered plugins
func (r *DomainRegistry) GetPlugins() map[string]DomainPlugin {
	return r.plugins
}

// DomainPluginRunner executes a plugin with the core orchestrator
type DomainPluginRunner struct {
	registry *DomainRegistry
	storage  domain.Storage
}

// NewDomainPluginRunner creates a new plugin runner
func NewDomainPluginRunner(registry *DomainRegistry, storage domain.Storage) *DomainPluginRunner {
	return &DomainPluginRunner{
		registry: registry,
		storage:  storage,
	}
}

// Execute runs a plugin with the given request
func (pr *DomainPluginRunner) Execute(ctx context.Context, pluginName, request string, opts ...DomainRunOption) error {
	plugin, err := pr.registry.Get(pluginName)
	if err != nil {
		return err
	}
	
	// Validate request for this plugin
	if err := plugin.ValidateRequest(request); err != nil {
		return &DomainInvalidRequestError{Plugin: pluginName, Reason: err.Error()}
	}
	
	// Get phases from plugin
	phases := plugin.GetPhases()
	
	// Execute phases sequentially with the core orchestrator
	input := domain.PhaseInput{
		Request: request,
		Data:    nil,
	}
	
	for i, phase := range phases {
		// Execute phase
		output, err := phase.Execute(ctx, input)
		if err != nil {
			return &DomainPhaseExecutionError{
				Plugin:    pluginName,
				Phase:     phase.Name(),
				Err:       err,
				Retryable: true, // TODO: Determine retryability based on error type
			}
		}
		
		// Validate output
		if err := phase.ValidateOutput(ctx, output); err != nil {
			return &DomainPhaseValidationError{
				Plugin: pluginName,
				Phase:  phase.Name(),
				Reason: err.Error(),
			}
		}
		
		// Save intermediate results
		if pr.storage != nil {
			// Convert output data to bytes for storage
			var dataBytes []byte
			if output.Data != nil {
				// TODO: Implement proper serialization based on data type
				// For now, convert to string representation
				dataBytes = []byte(fmt.Sprintf("%v", output.Data))
			}
			if err := pr.storage.Save(ctx, fmt.Sprintf("phase_%d_%s.json", i+1, phase.Name()), dataBytes); err != nil {
				// Log but don't fail on storage errors
				// TODO: Add logger to DomainPluginRunner
			}
		}
		
		// Pass output as input to next phase
		if i < len(phases)-1 {
			input = domain.PhaseInput{
				Request: request,
				Data:    output.Data,
			}
		}
	}
	
	return nil
}

// DomainRunOption configures the plugin execution
type DomainRunOption func(*DomainPluginRunner)

// WithDomainCheckpointing enables checkpoint support
func WithDomainCheckpointing(enabled bool) DomainRunOption {
	return func(pr *DomainPluginRunner) {
		// To be implemented
	}
}

// WithDomainMaxRetries sets the maximum retry count
func WithDomainMaxRetries(max int) DomainRunOption {
	return func(pr *DomainPluginRunner) {
		// To be implemented
	}
}