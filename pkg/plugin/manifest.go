// Package plugin provides plugin discovery and loading capabilities
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Manifest describes a plugin's metadata and capabilities
type Manifest struct {
	// Metadata
	Name        string    `json:"name" yaml:"name" validate:"required"`
	Version     string    `json:"version" yaml:"version" validate:"required,semver"`
	Description string    `json:"description" yaml:"description"`
	Author      string    `json:"author" yaml:"author"`
	License     string    `json:"license" yaml:"license"`
	Homepage    string    `json:"homepage" yaml:"homepage,omitempty"`
	Repository  string    `json:"repository" yaml:"repository,omitempty"`
	Tags        []string  `json:"tags" yaml:"tags,omitempty"`
	Created     time.Time `json:"created" yaml:"created"`
	Updated     time.Time `json:"updated" yaml:"updated"`

	// Plugin Type and Compatibility
	Type         PluginType `json:"type" yaml:"type" validate:"required,oneof=builtin external"`
	MinVersion   string     `json:"min_version" yaml:"min_version" validate:"omitempty,semver"`
	MaxVersion   string     `json:"max_version" yaml:"max_version" validate:"omitempty,semver"`
	Dependencies []string   `json:"dependencies" yaml:"dependencies,omitempty"`

	// Capabilities
	Domains      []string          `json:"domains" yaml:"domains" validate:"required,dive,oneof=fiction code docs"`
	Phases       []PhaseDefinition `json:"phases" yaml:"phases" validate:"required,dive"`
	Prompts      map[string]string `json:"prompts" yaml:"prompts"`
	OutputSpec   OutputSpec        `json:"output_spec" yaml:"output_spec"`
	ResourceSpec ResourceSpec      `json:"resource_spec" yaml:"resource_spec"`

	// Configuration
	ConfigSchema   json.RawMessage        `json:"config_schema,omitempty" yaml:"config_schema,omitempty"`
	DefaultConfig  map[string]interface{} `json:"default_config,omitempty" yaml:"default_config,omitempty"`
	RequiredConfig []string               `json:"required_config,omitempty" yaml:"required_config,omitempty"`

	// Location and Entry Point
	Location   string `json:"location" yaml:"location"`           // Directory path
	EntryPoint string `json:"entry_point" yaml:"entry_point"`     // Main file or binary
	Binary     bool   `json:"binary" yaml:"binary"`               // True if precompiled binary
	Language   string `json:"language" yaml:"language,omitempty"` // Programming language
}

// PluginType indicates whether a plugin is built-in or external
type PluginType string

const (
	PluginTypeBuiltin  PluginType = "builtin"
	PluginTypeExternal PluginType = "external"
)

// PhaseDefinition describes a phase provided by the plugin
type PhaseDefinition struct {
	Name            string            `json:"name" yaml:"name" validate:"required"`
	Description     string            `json:"description" yaml:"description"`
	Order           int               `json:"order" yaml:"order"`
	Required        bool              `json:"required" yaml:"required"`
	Parallel        bool              `json:"parallel" yaml:"parallel"`
	EstimatedTime   time.Duration     `json:"estimated_time" yaml:"estimated_time"`
	Timeout         time.Duration     `json:"timeout" yaml:"timeout"`
	Retryable       bool              `json:"retryable" yaml:"retryable"`
	MaxRetries      int               `json:"max_retries" yaml:"max_retries"`
	InputSchema     json.RawMessage   `json:"input_schema,omitempty" yaml:"input_schema,omitempty"`
	OutputSchema    json.RawMessage   `json:"output_schema,omitempty" yaml:"output_schema,omitempty"`
	ConfigOverrides map[string]string `json:"config_overrides,omitempty" yaml:"config_overrides,omitempty"`
}

// OutputSpec describes the expected outputs from a plugin
type OutputSpec struct {
	PrimaryOutput    string            `json:"primary_output" yaml:"primary_output"`
	SecondaryOutputs []string          `json:"secondary_outputs" yaml:"secondary_outputs"`
	FilePatterns     map[string]string `json:"file_patterns" yaml:"file_patterns"`
	Descriptions     map[string]string `json:"descriptions" yaml:"descriptions"`
}

// ResourceSpec defines resource requirements and limits
type ResourceSpec struct {
	MinMemory        string            `json:"min_memory,omitempty" yaml:"min_memory,omitempty"`
	MaxMemory        string            `json:"max_memory,omitempty" yaml:"max_memory,omitempty"`
	CPUShares        int               `json:"cpu_shares,omitempty" yaml:"cpu_shares,omitempty"`
	NetworkRequired  bool              `json:"network_required" yaml:"network_required"`
	StorageRequired  string            `json:"storage_required,omitempty" yaml:"storage_required,omitempty"`
	APIKeys          []string          `json:"api_keys,omitempty" yaml:"api_keys,omitempty"`
	EnvironmentVars  []string          `json:"environment_vars,omitempty" yaml:"environment_vars,omitempty"`
	Permissions      []string          `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	RateLimits       map[string]int    `json:"rate_limits,omitempty" yaml:"rate_limits,omitempty"`
}

// LoadManifest loads a plugin manifest from a file
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	manifest := &Manifest{}
	
	// Determine format based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, manifest); err != nil {
			return nil, fmt.Errorf("failed to parse YAML manifest: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, manifest); err != nil {
			return nil, fmt.Errorf("failed to parse JSON manifest: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, manifest); err != nil {
			if jsonErr := json.Unmarshal(data, manifest); jsonErr != nil {
				return nil, fmt.Errorf("failed to parse manifest as YAML or JSON: %w", err)
			}
		}
	}

	// Set location to the directory containing the manifest
	manifest.Location = filepath.Dir(path)

	// Validate the manifest
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return manifest, nil
}

// SaveManifest saves a plugin manifest to a file
func SaveManifest(manifest *Manifest, path string) error {
	// Update timestamp
	manifest.Updated = time.Now()
	if manifest.Created.IsZero() {
		manifest.Created = manifest.Updated
	}

	var data []byte
	var err error

	// Determine format based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(manifest)
	case ".json":
		data, err = json.MarshalIndent(manifest, "", "  ")
	default:
		// Default to YAML
		data, err = yaml.Marshal(manifest)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return nil
}

// Validate checks if the manifest is valid
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if m.Type == "" {
		return fmt.Errorf("plugin type is required")
	}
	if len(m.Domains) == 0 {
		return fmt.Errorf("at least one domain is required")
	}
	if len(m.Phases) == 0 {
		return fmt.Errorf("at least one phase is required")
	}

	// Validate domain names
	validDomains := map[string]bool{"fiction": true, "code": true, "docs": true}
	for _, domain := range m.Domains {
		if !validDomains[domain] {
			return fmt.Errorf("invalid domain: %s", domain)
		}
	}

	// Validate phase definitions
	phaseNames := make(map[string]bool)
	for i, phase := range m.Phases {
		if phase.Name == "" {
			return fmt.Errorf("phase[%d] name is required", i)
		}
		if phaseNames[phase.Name] {
			return fmt.Errorf("duplicate phase name: %s", phase.Name)
		}
		phaseNames[phase.Name] = true
	}

	return nil
}

// IsCompatible checks if the plugin is compatible with the given orchestrator version
func (m *Manifest) IsCompatible(version string) bool {
	// TODO: Implement semantic version comparison
	// For now, always return true
	return true
}

// GetPhase returns a phase definition by name
func (m *Manifest) GetPhase(name string) (*PhaseDefinition, bool) {
	for i := range m.Phases {
		if m.Phases[i].Name == name {
			return &m.Phases[i], true
		}
	}
	return nil, false
}

// GetPromptPath returns the full path to a prompt file
func (m *Manifest) GetPromptPath(phaseName string) string {
	if promptFile, ok := m.Prompts[phaseName]; ok {
		if filepath.IsAbs(promptFile) {
			return promptFile
		}
		return filepath.Join(m.Location, promptFile)
	}
	return ""
}

// GetConfigValue retrieves a configuration value with fallback to defaults
func (m *Manifest) GetConfigValue(key string) (interface{}, bool) {
	if val, ok := m.DefaultConfig[key]; ok {
		return val, true
	}
	return nil, false
}

// IsRequired checks if a configuration key is required
func (m *Manifest) IsRequired(key string) bool {
	for _, req := range m.RequiredConfig {
		if req == key {
			return true
		}
	}
	return false
}

// String returns a string representation of the manifest
func (m *Manifest) String() string {
	return fmt.Sprintf("%s v%s (%s)", m.Name, m.Version, m.Type)
}