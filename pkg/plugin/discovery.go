package plugin

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Discoverer finds and catalogs available plugins
type Discoverer struct {
	logger       *slog.Logger
	searchPaths  []string
	manifestName string
	cache        *discoveryCache
}

// discoveryCache stores discovered plugins to avoid re-scanning
type discoveryCache struct {
	mu       sync.RWMutex
	plugins  map[string]*Manifest
	lastScan map[string]int64 // path -> modification time
}

// NewDiscoverer creates a new plugin discoverer
func NewDiscoverer(logger *slog.Logger) *Discoverer {
	return &Discoverer{
		logger:       logger,
		searchPaths:  getDefaultSearchPaths(),
		manifestName: "plugin.yaml",
		cache: &discoveryCache{
			plugins:  make(map[string]*Manifest),
			lastScan: make(map[string]int64),
		},
	}
}

// getDefaultSearchPaths returns XDG-compliant plugin search paths
func getDefaultSearchPaths() []string {
	paths := []string{}

	// Built-in plugins (relative to binary location)
	if execPath, err := os.Executable(); err == nil {
		binDir := filepath.Dir(execPath)
		paths = append(paths, filepath.Join(binDir, "..", "share", "orchestrator", "plugins"))
	}

	// XDG data directories
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}
	if dataHome != "" {
		paths = append(paths, filepath.Join(dataHome, "orchestrator", "plugins"))
	}

	// System-wide locations
	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	for _, dir := range filepath.SplitList(dataDirs) {
		if dir != "" {
			paths = append(paths, filepath.Join(dir, "orchestrator", "plugins"))
		}
	}

	// User config directory (for user-installed plugins)
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			configHome = filepath.Join(home, ".config")
		}
	}
	if configHome != "" {
		paths = append(paths, filepath.Join(configHome, "orchestrator", "plugins"))
	}

	// Development/local plugins
	if pwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(pwd, "plugins"))
	}

	return paths
}

// SetSearchPaths overrides the default search paths
func (d *Discoverer) SetSearchPaths(paths []string) {
	d.searchPaths = paths
	// Clear cache when paths change
	d.cache.mu.Lock()
	d.cache.plugins = make(map[string]*Manifest)
	d.cache.lastScan = make(map[string]int64)
	d.cache.mu.Unlock()
}

// AddSearchPath adds a path to search for plugins
func (d *Discoverer) AddSearchPath(path string) {
	d.searchPaths = append(d.searchPaths, path)
}

// SetManifestName changes the manifest filename to look for
func (d *Discoverer) SetManifestName(name string) {
	d.manifestName = name
}

// Discover finds all available plugins
func (d *Discoverer) Discover() ([]*Manifest, error) {
	d.logger.Debug("discovering plugins", "search_paths", d.searchPaths)

	allPlugins := make(map[string]*Manifest) // name -> manifest
	var errors []error

	for _, searchPath := range d.searchPaths {
		// Check if path exists
		info, err := os.Stat(searchPath)
		if err != nil {
			if !os.IsNotExist(err) {
				d.logger.Debug("error accessing search path", "path", searchPath, "error", err)
			}
			continue
		}

		// Check cache
		d.cache.mu.RLock()
		lastMod, cached := d.cache.lastScan[searchPath]
		d.cache.mu.RUnlock()

		if cached && info.ModTime().Unix() <= lastMod {
			// Use cached results for this path
			d.cache.mu.RLock()
			for name, manifest := range d.cache.plugins {
				if strings.HasPrefix(manifest.Location, searchPath) {
					allPlugins[name] = manifest
				}
			}
			d.cache.mu.RUnlock()
			continue
		}

		// Scan directory
		plugins, err := d.scanDirectory(searchPath)
		if err != nil {
			errors = append(errors, fmt.Errorf("error scanning %s: %w", searchPath, err))
			continue
		}

		// Update cache
		d.cache.mu.Lock()
		d.cache.lastScan[searchPath] = info.ModTime().Unix()
		for _, plugin := range plugins {
			d.cache.plugins[plugin.Name] = plugin
			allPlugins[plugin.Name] = plugin
		}
		d.cache.mu.Unlock()
	}

	// Convert map to sorted slice
	result := make([]*Manifest, 0, len(allPlugins))
	for _, manifest := range allPlugins {
		result = append(result, manifest)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	d.logger.Info("plugin discovery complete", "found", len(result))
	return result, nil
}

// scanDirectory scans a directory for plugin manifests
func (d *Discoverer) scanDirectory(dir string) ([]*Manifest, error) {
	var plugins []*Manifest

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible directories
		}

		// Skip if not a manifest file
		if entry.IsDir() || !d.isManifestFile(entry.Name()) {
			return nil
		}

		// Load manifest
		manifest, err := LoadManifest(path)
		if err != nil {
			d.logger.Warn("failed to load manifest", "path", path, "error", err)
			return nil // Continue scanning
		}

		// Validate plugin location
		pluginDir := filepath.Dir(path)
		if manifest.EntryPoint != "" {
			entryPath := filepath.Join(pluginDir, manifest.EntryPoint)
			if _, err := os.Stat(entryPath); err != nil {
				d.logger.Warn("plugin entry point not found", 
					"plugin", manifest.Name,
					"entry_point", entryPath,
					"error", err)
				return nil
			}
		}

		d.logger.Debug("discovered plugin", 
			"name", manifest.Name,
			"version", manifest.Version,
			"type", manifest.Type,
			"location", pluginDir)

		plugins = append(plugins, manifest)
		return nil
	})

	return plugins, err
}

// isManifestFile checks if a filename is a plugin manifest
func (d *Discoverer) isManifestFile(name string) bool {
	// Check exact match
	if name == d.manifestName {
		return true
	}

	// Check common variations
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "plugin" || base == "manifest" {
		ext := filepath.Ext(name)
		return ext == ".yaml" || ext == ".yml" || ext == ".json"
	}

	return false
}

// DiscoverByDomain finds plugins that support a specific domain
func (d *Discoverer) DiscoverByDomain(domain string) ([]*Manifest, error) {
	allPlugins, err := d.Discover()
	if err != nil {
		return nil, err
	}

	var filtered []*Manifest
	for _, plugin := range allPlugins {
		for _, d := range plugin.Domains {
			if d == domain {
				filtered = append(filtered, plugin)
				break
			}
		}
	}

	return filtered, nil
}

// DiscoverByType finds plugins of a specific type
func (d *Discoverer) DiscoverByType(pluginType PluginType) ([]*Manifest, error) {
	allPlugins, err := d.Discover()
	if err != nil {
		return nil, err
	}

	var filtered []*Manifest
	for _, plugin := range allPlugins {
		if plugin.Type == pluginType {
			filtered = append(filtered, plugin)
		}
	}

	return filtered, nil
}

// GetPlugin finds a specific plugin by name
func (d *Discoverer) GetPlugin(name string) (*Manifest, error) {
	// Check cache first
	d.cache.mu.RLock()
	if manifest, ok := d.cache.plugins[name]; ok {
		d.cache.mu.RUnlock()
		return manifest, nil
	}
	d.cache.mu.RUnlock()

	// Do a fresh discovery
	plugins, err := d.Discover()
	if err != nil {
		return nil, err
	}

	for _, plugin := range plugins {
		if plugin.Name == name {
			return plugin, nil
		}
	}

	return nil, fmt.Errorf("plugin not found: %s", name)
}

// ClearCache forces a fresh discovery on next call
func (d *Discoverer) ClearCache() {
	d.cache.mu.Lock()
	defer d.cache.mu.Unlock()
	d.cache.plugins = make(map[string]*Manifest)
	d.cache.lastScan = make(map[string]int64)
}

// PluginInfo provides a summary of discovered plugins
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Type        PluginType
	Domains     []string
	Location    string
}

// ListPlugins returns a summary of all discovered plugins
func (d *Discoverer) ListPlugins() ([]PluginInfo, error) {
	plugins, err := d.Discover()
	if err != nil {
		return nil, err
	}

	infos := make([]PluginInfo, len(plugins))
	for i, plugin := range plugins {
		infos[i] = PluginInfo{
			Name:        plugin.Name,
			Version:     plugin.Version,
			Description: plugin.Description,
			Type:        plugin.Type,
			Domains:     plugin.Domains,
			Location:    plugin.Location,
		}
	}

	return infos, nil
}