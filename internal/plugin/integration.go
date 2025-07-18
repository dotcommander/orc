package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"time"

	"github.com/dotcommander/orc/internal/agent"
	"github.com/dotcommander/orc/internal/config"
	"github.com/dotcommander/orc/internal/domain"
	domainPlugin "github.com/dotcommander/orc/internal/domain/plugin"
)

// PluginIntegrator manages both domain (built-in) and external plugins
type PluginIntegrator struct {
	config         *config.Config
	domainRegistry *domainPlugin.DomainRegistry
	externalPlugins map[string]ExternalPlugin
	logger         *slog.Logger
}

// ExternalPlugin represents a dynamically loaded plugin
type ExternalPlugin struct {
	Name        string
	Path        string
	Plugin      *plugin.Plugin
	Phases      []domain.Phase
	Description string
	Loaded      time.Time
}

// PluginDiscoveryResult contains information about discovered plugins
type PluginDiscoveryResult struct {
	DomainPlugins   []string
	ExternalPlugins []ExternalPluginInfo
	Errors          []PluginError
}

// ExternalPluginInfo contains metadata about an external plugin
type ExternalPluginInfo struct {
	Name        string
	Path        string
	Description string
	Version     string
	Compatible  bool
	Error       string
}

// PluginError represents plugin-related errors
type PluginError struct {
	Plugin string
	Path   string
	Error  string
}

// NewPluginIntegrator creates a new plugin integrator
func NewPluginIntegrator(cfg *config.Config, logger *slog.Logger) *PluginIntegrator {
	return &PluginIntegrator{
		config:          cfg,
		domainRegistry:  domainPlugin.NewDomainRegistry(),
		externalPlugins: make(map[string]ExternalPlugin),
		logger:          logger,
	}
}

// InitializeBuiltinPlugins now only logs that built-in plugins are deprecated
func (pi *PluginIntegrator) InitializeBuiltinPlugins(domainAgent domain.Agent, storage domain.Storage, promptsDir string, aiClient agent.AIClient) error {
	pi.logger.Info("Built-in plugins are deprecated. Use external plugins in plugins/ directory")
	
	// TODO: Remove this method entirely once migration is complete
	// For now, we'll try to load fiction and code as external plugins
	
	pi.logger.Info("Attempting to load fiction and code as external plugins")
	
	return nil
}

// DiscoverExternalPlugins searches for external plugins in configured paths
func (pi *PluginIntegrator) DiscoverExternalPlugins(ctx context.Context) (*PluginDiscoveryResult, error) {
	if !pi.config.Plugins.Settings.AutoDiscovery {
		pi.logger.Info("Plugin auto-discovery is disabled")
		return &PluginDiscoveryResult{
			DomainPlugins: pi.domainRegistry.List(),
		}, nil
	}
	
	result := &PluginDiscoveryResult{
		DomainPlugins:   pi.domainRegistry.List(),
		ExternalPlugins: []ExternalPluginInfo{},
		Errors:          []PluginError{},
	}
	
	pi.logger.Info("Discovering external plugins", 
		"paths", pi.config.Plugins.DiscoveryPaths,
		"max_plugins", pi.config.Plugins.Settings.MaxExternalPlugins)
	
	for _, searchPath := range pi.config.Plugins.DiscoveryPaths {
		if err := pi.discoverPluginsInPath(ctx, searchPath, result); err != nil {
			pi.logger.Warn("Error discovering plugins in path", 
				"path", searchPath, 
				"error", err)
			result.Errors = append(result.Errors, PluginError{
				Path:  searchPath,
				Error: err.Error(),
			})
		}
	}
	
	pi.logger.Info("Plugin discovery completed", 
		"domain_plugins", len(result.DomainPlugins),
		"external_plugins", len(result.ExternalPlugins),
		"errors", len(result.Errors))
	
	return result, nil
}

// discoverPluginsInPath searches for plugins in a specific directory
func (pi *PluginIntegrator) discoverPluginsInPath(ctx context.Context, searchPath string, result *PluginDiscoveryResult) error {
	// Check if path exists
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		pi.logger.Debug("Plugin path does not exist", "path", searchPath)
		return nil
	}
	
	// Walk the directory looking for plugin files
	return filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}
		
		// Skip non-plugin files
		if !pi.isPluginFile(path, info) {
			return nil
		}
		
		// Check if we've reached the maximum number of external plugins
		if len(result.ExternalPlugins) >= pi.config.Plugins.Settings.MaxExternalPlugins {
			pi.logger.Warn("Maximum external plugins reached", 
				"max", pi.config.Plugins.Settings.MaxExternalPlugins)
			return filepath.SkipDir
		}
		
		// Analyze the plugin
		pluginInfo := pi.analyzeExternalPlugin(ctx, path)
		result.ExternalPlugins = append(result.ExternalPlugins, pluginInfo)
		
		// Load the plugin if it's compatible and enabled
		if pluginInfo.Compatible && pi.isPluginEnabled(pluginInfo.Name) {
			if err := pi.loadExternalPlugin(ctx, path, pluginInfo.Name); err != nil {
				pi.logger.Warn("Failed to load external plugin", 
					"path", path, 
					"error", err)
				result.Errors = append(result.Errors, PluginError{
					Plugin: pluginInfo.Name,
					Path:   path,
					Error:  err.Error(),
				})
			}
		}
		
		return nil
	})
}

// isPluginFile determines if a file is a potential plugin
func (pi *PluginIntegrator) isPluginFile(path string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	
	// Look for shared objects (.so files) or executable binaries
	ext := filepath.Ext(path)
	if ext == ".so" {
		return true
	}
	
	// Check if it's an executable with "orc-" prefix
	basename := filepath.Base(path)
	if (info.Mode()&0111) != 0 && // executable
		(filepath.HasPrefix(basename, "orc-") || filepath.HasPrefix(basename, "orchestrator-")) {
		return true
	}
	
	return false
}

// analyzeExternalPlugin examines a plugin file without loading it
func (pi *PluginIntegrator) analyzeExternalPlugin(ctx context.Context, path string) ExternalPluginInfo {
	basename := filepath.Base(path)
	name := pi.extractPluginName(basename)
	
	info := ExternalPluginInfo{
		Name:        name,
		Path:        path,
		Description: "External plugin",
		Version:     "unknown",
		Compatible:  false,
	}
	
	// Check if it's a shared object plugin
	if filepath.Ext(path) == ".so" {
		info.Compatible = pi.isSharedObjectCompatible(path)
		if info.Compatible {
			info.Description = "Go shared object plugin"
		} else {
			info.Error = "Incompatible shared object format"
		}
		return info
	}
	
	// For executable plugins, check if they support the plugin protocol
	if pi.isExecutablePlugin(path) {
		info.Compatible = true
		info.Description = "Executable plugin"
		// TODO: Query executable for metadata
	} else {
		info.Error = "Not a valid plugin executable"
	}
	
	return info
}

// extractPluginName extracts the plugin name from filename
func (pi *PluginIntegrator) extractPluginName(filename string) string {
	name := filename
	
	// Remove common prefixes
	if filepath.HasPrefix(name, "orc-") {
		name = name[4:]
	} else if filepath.HasPrefix(name, "orchestrator-") {
		name = name[13:]
	}
	
	// Remove extension
	if ext := filepath.Ext(name); ext != "" {
		name = name[:len(name)-len(ext)]
	}
	
	return name
}

// isSharedObjectCompatible checks if a .so file is compatible
func (pi *PluginIntegrator) isSharedObjectCompatible(path string) bool {
	// Try to open the plugin to verify compatibility
	p, err := plugin.Open(path)
	if err != nil {
		return false
	}
	
	// Check for required plugin interface symbols
	if _, err := p.Lookup("GetPlugin"); err != nil {
		return false
	}
	
	return true
}

// isExecutablePlugin checks if an executable supports the plugin protocol
func (pi *PluginIntegrator) isExecutablePlugin(path string) bool {
	// Check if file is executable
	if info, err := os.Stat(path); err != nil || (info.Mode()&0111) == 0 {
		return false
	}
	
	// TODO: Execute with --plugin-info flag to check protocol support
	// For now, assume executables with correct naming are plugins
	basename := filepath.Base(path)
	return filepath.HasPrefix(basename, "orc-") || filepath.HasPrefix(basename, "orchestrator-")
}

// isPluginEnabled checks if a plugin is enabled in configuration
func (pi *PluginIntegrator) isPluginEnabled(name string) bool {
	if config, exists := pi.config.Plugins.Configurations[name]; exists {
		return config.Enabled
	}
	// Default to enabled if not explicitly configured
	return true
}

// loadExternalPlugin loads an external plugin
func (pi *PluginIntegrator) loadExternalPlugin(ctx context.Context, path, name string) error {
	pi.logger.Info("Loading external plugin", "name", name, "path", path)
	
	if filepath.Ext(path) == ".so" {
		return pi.loadSharedObjectPlugin(ctx, path, name)
	}
	
	return pi.loadExecutablePlugin(ctx, path, name)
}

// loadSharedObjectPlugin loads a Go shared object plugin
func (pi *PluginIntegrator) loadSharedObjectPlugin(ctx context.Context, path, name string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}
	
	// Look up the plugin interface
	getPluginSym, err := p.Lookup("GetPlugin")
	if err != nil {
		return fmt.Errorf("plugin missing GetPlugin function: %w", err)
	}
	
	getPlugin, ok := getPluginSym.(func() interface{})
	if !ok {
		return fmt.Errorf("GetPlugin has wrong signature")
	}
	
	// Get the plugin implementation
	_ = getPlugin() // TODO: Convert plugin implementation to domain.Phase interface
	                // This requires defining a plugin interface protocol
	
	extPlugin := ExternalPlugin{
		Name:        name,
		Path:        path,
		Plugin:      p,
		Phases:      []domain.Phase{}, // TODO: Extract phases from plugin
		Description: "Loaded shared object plugin",
		Loaded:      time.Now(),
	}
	
	pi.externalPlugins[name] = extPlugin
	pi.logger.Info("Successfully loaded shared object plugin", "name", name)
	
	return nil
}

// loadExecutablePlugin loads an executable plugin
func (pi *PluginIntegrator) loadExecutablePlugin(ctx context.Context, path, name string) error {
	// TODO: Implement executable plugin loading
	// This would involve process management and IPC
	
	extPlugin := ExternalPlugin{
		Name:        name,
		Path:        path,
		Plugin:      nil, // No Go plugin object for executables
		Phases:      []domain.Phase{}, // TODO: Create phases that communicate with executable
		Description: "Executable plugin",
		Loaded:      time.Now(),
	}
	
	pi.externalPlugins[name] = extPlugin
	pi.logger.Info("Successfully registered executable plugin", "name", name)
	
	return nil
}

// GetAvailablePlugins returns all available plugins (domain + external)
func (pi *PluginIntegrator) GetAvailablePlugins() map[string]interface{} {
	plugins := make(map[string]interface{})
	
	// Add domain plugins
	for name, plugin := range pi.domainRegistry.GetPlugins() {
		plugins[name] = plugin
	}
	
	// Add external plugins
	for name, plugin := range pi.externalPlugins {
		plugins[name] = plugin
	}
	
	return plugins
}

// GetPlugin retrieves a plugin by name (domain or external)
func (pi *PluginIntegrator) GetPlugin(name string) (interface{}, error) {
	// First check domain plugins
	if domainPlugin, err := pi.domainRegistry.Get(name); err == nil {
		return domainPlugin, nil
	}
	
	// Then check external plugins
	if extPlugin, exists := pi.externalPlugins[name]; exists {
		return extPlugin, nil
	}
	
	return nil, fmt.Errorf("plugin '%s' not found", name)
}

// GetDomainRegistry returns the domain plugin registry
func (pi *PluginIntegrator) GetDomainRegistry() *domainPlugin.DomainRegistry {
	return pi.domainRegistry
}

// GetExternalPlugins returns the external plugins
func (pi *PluginIntegrator) GetExternalPlugins() map[string]ExternalPlugin {
	return pi.externalPlugins
}

// CreatePluginDirectories ensures plugin directories exist
func (pi *PluginIntegrator) CreatePluginDirectories() error {
	directories := []string{
		pi.config.Plugins.BuiltinPath,
		pi.config.Plugins.ExternalPath,
	}
	
	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create plugin directory %s: %w", dir, err)
		}
		pi.logger.Debug("Created plugin directory", "path", dir)
	}
	
	return nil
}