package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"github.com/dotcommander/orc/internal/domain"
	domainPlugin "github.com/dotcommander/orc/internal/domain/plugin"
)

// Loader loads and manages plugins
type Loader struct {
	logger     *slog.Logger
	discoverer *Discoverer
	registry   *domainPlugin.DomainRegistry
	loaded     map[string]LoadedPlugin
	mu         sync.RWMutex
}

// LoadedPlugin represents a loaded plugin instance
type LoadedPlugin struct {
	Manifest *Manifest
	Plugin   domainPlugin.DomainPlugin
	LoadedAt time.Time
	Handle   interface{} // For external Go plugins
	Process  *exec.Cmd      // For external binary plugins
}

// NewLoader creates a new plugin loader
func NewLoader(logger *slog.Logger, discoverer *Discoverer, registry *domainPlugin.DomainRegistry) *Loader {
	return &Loader{
		logger:     logger,
		discoverer: discoverer,
		registry:   registry,
		loaded:     make(map[string]LoadedPlugin),
	}
}

// LoadAll loads all discovered plugins
func (l *Loader) LoadAll() error {
	manifests, err := l.discoverer.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	var loadErrors []error
	for _, manifest := range manifests {
		if err := l.Load(manifest); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to load %s: %w", manifest.Name, err))
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("some plugins failed to load: %v", loadErrors)
	}

	return nil
}

// Load loads a specific plugin
func (l *Loader) Load(manifest *Manifest) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if _, exists := l.loaded[manifest.Name]; exists {
		l.logger.Debug("plugin already loaded", "name", manifest.Name)
		return nil
	}

	l.logger.Info("loading plugin", "name", manifest.Name, "type", manifest.Type)

	var domainPlg domainPlugin.DomainPlugin
	var handle interface{}
	var process *exec.Cmd
	var err error

	switch manifest.Type {
	case PluginTypeBuiltin:
		domainPlg, err = l.loadBuiltinPlugin(manifest)
	case PluginTypeExternal:
		if manifest.Binary {
			domainPlg, process, err = l.loadBinaryPlugin(manifest)
		} else {
			domainPlg, handle, err = l.loadGoPlugin(manifest)
		}
	default:
		return fmt.Errorf("unsupported plugin type: %s", manifest.Type)
	}

	if err != nil {
		return err
	}

	// Register with domain registry
	if err := l.registry.Register(domainPlg); err != nil {
		// If plugin already exists, try to replace it
		l.registry.Replace(domainPlg)
	}

	// Store loaded plugin
	l.loaded[manifest.Name] = LoadedPlugin{
		Manifest: manifest,
		Plugin:   domainPlg,
		LoadedAt: time.Now(),
		Handle:   handle,
		Process:  process,
	}

	l.logger.Info("plugin loaded successfully", "name", manifest.Name)
	return nil
}

// loadBuiltinPlugin loads a built-in plugin (already compiled into the binary)
func (l *Loader) loadBuiltinPlugin(manifest *Manifest) (domainPlugin.DomainPlugin, error) {
	// Built-in plugins are registered at compile time
	// Try to get from registry
	plugin, err := l.registry.Get(manifest.Name)
	if err != nil {
		return nil, fmt.Errorf("built-in plugin not found in registry: %s", manifest.Name)
	}
	return plugin, nil
}

// loadGoPlugin loads an external Go plugin (.so file)
func (l *Loader) loadGoPlugin(manifest *Manifest) (domainPlugin.DomainPlugin, interface{}, error) {
	if manifest.EntryPoint == "" {
		return nil, nil, fmt.Errorf("no entry point specified for plugin %s", manifest.Name)
	}

	pluginPath := filepath.Join(manifest.Location, manifest.EntryPoint)
	
	// Ensure it's a .so file
	if !strings.HasSuffix(pluginPath, ".so") {
		pluginPath += ".so"
	}

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the Plugin symbol
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return nil, nil, fmt.Errorf("plugin missing 'Plugin' symbol: %w", err)
	}

	// Assert the type
	domainPlg, ok := symPlugin.(domainPlugin.DomainPlugin)
	if !ok {
		// Try pointer type
		if ptr, ok := symPlugin.(*domainPlugin.DomainPlugin); ok {
			domainPlg = *ptr
		} else {
			return nil, nil, fmt.Errorf("plugin 'Plugin' symbol has wrong type: %T", symPlugin)
		}
	}

	return domainPlg, p, nil
}

// loadBinaryPlugin loads an external binary plugin (executable)
func (l *Loader) loadBinaryPlugin(manifest *Manifest) (domainPlugin.DomainPlugin, *exec.Cmd, error) {
	if manifest.EntryPoint == "" {
		return nil, nil, fmt.Errorf("no entry point specified for plugin %s", manifest.Name)
	}

	execPath := filepath.Join(manifest.Location, manifest.EntryPoint)

	// Create a wrapper that communicates with the binary via JSON-RPC or similar
	wrapper := &binaryPluginWrapper{
		manifest: manifest,
		execPath: execPath,
		logger:   l.logger,
	}

	// Start the plugin process (if needed)
	// For now, we'll start processes on-demand in the wrapper

	return wrapper, nil, nil
}

// binaryPluginWrapper wraps an external binary plugin
type binaryPluginWrapper struct {
	manifest *Manifest
	execPath string
	logger   *slog.Logger
}

// Implement DomainPlugin interface
func (w *binaryPluginWrapper) Name() string {
	return w.manifest.Name
}

func (w *binaryPluginWrapper) Description() string {
	return w.manifest.Description
}

func (w *binaryPluginWrapper) GetPhases() []domain.Phase {
	// Convert manifest phases to domain phases
	phases := make([]domain.Phase, len(w.manifest.Phases))
	for i, phaseDef := range w.manifest.Phases {
		phases[i] = &binaryPhaseWrapper{
			wrapper:    w,
			definition: phaseDef,
		}
	}
	return phases
}

func (w *binaryPluginWrapper) GetDefaultConfig() domainPlugin.DomainPluginConfig {
	config := domainPlugin.DomainPluginConfig{
		Prompts:  w.manifest.Prompts,
		Metadata: w.manifest.DefaultConfig,
	}

	// Convert resource spec to limits
	if w.manifest.ResourceSpec.MaxMemory != "" || len(w.manifest.Phases) > 0 {
		config.Limits = domainPlugin.DomainPluginLimits{
			MaxConcurrentPhases: 1, // Binary plugins run sequentially by default
			PhaseTimeouts:       make(map[string]time.Duration),
			MaxRetries:          3,
			TotalTimeout:        30 * time.Minute,
		}

		// Set phase timeouts from manifest
		for _, phase := range w.manifest.Phases {
			if phase.Timeout > 0 {
				config.Limits.PhaseTimeouts[phase.Name] = phase.Timeout
			}
		}
	}

	return config
}

func (w *binaryPluginWrapper) ValidateRequest(request string) error {
	// Basic validation - can be enhanced
	if request == "" {
		return fmt.Errorf("empty request")
	}
	return nil
}

func (w *binaryPluginWrapper) GetOutputSpec() domainPlugin.DomainOutputSpec {
	return domainPlugin.DomainOutputSpec{
		PrimaryOutput:    w.manifest.OutputSpec.PrimaryOutput,
		SecondaryOutputs: w.manifest.OutputSpec.SecondaryOutputs,
		Descriptions:     w.manifest.OutputSpec.Descriptions,
	}
}

func (w *binaryPluginWrapper) GetDomainValidator() domain.DomainValidator {
	// Return a basic validator
	return &basicDomainValidator{
		domains: w.manifest.Domains,
	}
}

// binaryPhaseWrapper wraps a phase from a binary plugin
type binaryPhaseWrapper struct {
	wrapper    *binaryPluginWrapper
	definition PhaseDefinition
}

func (p *binaryPhaseWrapper) Name() string {
	return p.definition.Name
}

func (p *binaryPhaseWrapper) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Execute the binary with phase name and input
	cmd := exec.CommandContext(ctx, p.wrapper.execPath, "execute", p.definition.Name)

	// Pass input as JSON via stdin
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return domain.PhaseOutput{}, fmt.Errorf("failed to marshal input: %w", err)
	}

	cmd.Stdin = strings.NewReader(string(inputJSON))

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PLUGIN_NAME=%s", p.wrapper.manifest.Name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PHASE_NAME=%s", p.definition.Name))

	// Execute and capture output
	output, err := cmd.Output()
	if err != nil {
		return domain.PhaseOutput{}, fmt.Errorf("phase execution failed: %w", err)
	}

	// Parse output as JSON
	var phaseOutput domain.PhaseOutput
	if err := json.Unmarshal(output, &phaseOutput); err != nil {
		return domain.PhaseOutput{}, fmt.Errorf("failed to parse phase output: %w", err)
	}

	return phaseOutput, nil
}

func (p *binaryPhaseWrapper) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	// Basic validation
	if p.definition.Required && input.Request == "" {
		return fmt.Errorf("required phase %s received empty request", p.Name())
	}
	return nil
}

func (p *binaryPhaseWrapper) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	// Basic output validation
	if output.Data == nil && p.definition.Required {
		return fmt.Errorf("required phase %s produced no output", p.Name())
	}
	return nil
}

func (p *binaryPhaseWrapper) EstimatedDuration() time.Duration {
	if p.definition.EstimatedTime > 0 {
		return p.definition.EstimatedTime
	}
	return 5 * time.Minute // Default estimate
}

func (p *binaryPhaseWrapper) CanRetry(err error) bool {
	return p.definition.Retryable
}

// basicDomainValidator provides basic domain validation
type basicDomainValidator struct {
	domains []string
}

func (v *basicDomainValidator) ValidateRequest(request string) error {
	if request == "" {
		return fmt.Errorf("empty request")
	}
	return nil
}

func (v *basicDomainValidator) ValidatePhaseTransition(from, to string, data interface{}) error {
	// Basic validation - no specific transition rules
	return nil
}

// Unload unloads a specific plugin
func (l *Loader) Unload(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	loaded, exists := l.loaded[name]
	if !exists {
		return fmt.Errorf("plugin not loaded: %s", name)
	}

	// Stop any running processes
	if loaded.Process != nil {
		if err := loaded.Process.Process.Kill(); err != nil {
			l.logger.Warn("failed to kill plugin process", "plugin", name, "error", err)
		}
	}

	// Remove from registry
	// Note: registry doesn't have a Remove method in current implementation
	// This would need to be added

	// Remove from loaded map
	delete(l.loaded, name)

	l.logger.Info("plugin unloaded", "name", name)
	return nil
}

// UnloadAll unloads all plugins
func (l *Loader) UnloadAll() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for name := range l.loaded {
		if err := l.Unload(name); err != nil {
			l.logger.Error("failed to unload plugin", "name", name, "error", err)
		}
	}
}

// GetLoaded returns all loaded plugins
func (l *Loader) GetLoaded() map[string]LoadedPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	copy := make(map[string]LoadedPlugin)
	for k, v := range l.loaded {
		copy[k] = v
	}
	return copy
}

// IsLoaded checks if a plugin is loaded
func (l *Loader) IsLoaded(name string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, exists := l.loaded[name]
	return exists
}

// Reload reloads a plugin
func (l *Loader) Reload(name string) error {
	// Get the manifest
	manifest, err := l.discoverer.GetPlugin(name)
	if err != nil {
		return fmt.Errorf("failed to get plugin manifest: %w", err)
	}

	// Unload if already loaded
	if l.IsLoaded(name) {
		if err := l.Unload(name); err != nil {
			return fmt.Errorf("failed to unload plugin: %w", err)
		}
	}

	// Load again
	return l.Load(manifest)
}

// ValidatePlugin validates a plugin without loading it
func (l *Loader) ValidatePlugin(manifest *Manifest) error {
	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Check entry point exists
	if manifest.EntryPoint != "" {
		entryPath := filepath.Join(manifest.Location, manifest.EntryPoint)
		if _, err := os.Stat(entryPath); err != nil {
			return fmt.Errorf("entry point not found: %w", err)
		}
	}

	// Validate prompts exist
	for phase := range manifest.Prompts {
		promptPath := manifest.GetPromptPath(phase)
		if promptPath != "" {
			if _, err := os.Stat(promptPath); err != nil {
				return fmt.Errorf("prompt file for phase %s not found: %w", phase, err)
			}
		}
	}

	return nil
}