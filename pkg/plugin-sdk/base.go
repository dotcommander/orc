// Package sdk provides utilities for building Orc plugins
package sdk

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotcommander/orc/pkg/orc"
)

// BasePhase provides common functionality for phases
type BasePhase struct {
	name              string
	estimatedDuration time.Duration
	canRetry          bool
}

// NewBasePhase creates a new base phase
func NewBasePhase(name string, duration time.Duration) BasePhase {
	return BasePhase{
		name:              name,
		estimatedDuration: duration,
		canRetry:          true,
	}
}

// Name returns the phase name
func (p *BasePhase) Name() string {
	return p.name
}

// EstimatedDuration returns the estimated duration
func (p *BasePhase) EstimatedDuration() time.Duration {
	return p.estimatedDuration
}

// CanRetry indicates if the phase can be retried
func (p *BasePhase) CanRetry(err error) bool {
	return p.canRetry
}

// ValidateInput provides default input validation
func (p *BasePhase) ValidateInput(ctx context.Context, input orc.PhaseInput) error {
	if input.Request == "" {
		return orc.ErrInvalidInput
	}
	return nil
}

// ValidateOutput provides default output validation
func (p *BasePhase) ValidateOutput(ctx context.Context, output orc.PhaseOutput) error {
	if output.Error != nil {
		return output.Error
	}
	return nil
}

// BasePlugin provides common functionality for plugins
type BasePlugin struct {
	name        string
	version     string
	description string
	author      string
	domains     []string
	phases      []orc.Phase
	logger      *slog.Logger
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name, version, description, author string, domains []string) BasePlugin {
	return BasePlugin{
		name:        name,
		version:     version,
		description: description,
		author:      author,
		domains:     domains,
		phases:      []orc.Phase{},
	}
}

// GetInfo returns plugin information
func (p *BasePlugin) GetInfo() orc.PluginInfo {
	return orc.PluginInfo{
		Name:          p.name,
		Version:       p.version,
		Description:   p.description,
		Author:        p.author,
		Domains:       p.domains,
		MinOrcVersion: "0.1.0",
	}
}

// CreatePhases returns the plugin's phases
func (p *BasePlugin) CreatePhases() ([]orc.Phase, error) {
	return p.phases, nil
}

// SetPhases sets the plugin's phases
func (p *BasePlugin) SetPhases(phases []orc.Phase) {
	p.phases = phases
}

// ValidateRequest provides default request validation
func (p *BasePlugin) ValidateRequest(request string) error {
	if len(request) < 10 {
		return orc.ErrInvalidInput
	}
	return nil
}

// GetPhaseTimeouts returns default timeouts
func (p *BasePlugin) GetPhaseTimeouts() map[string]time.Duration {
	timeouts := make(map[string]time.Duration)
	for _, phase := range p.phases {
		timeouts[phase.Name()] = phase.EstimatedDuration()
	}
	return timeouts
}

// GetRequiredConfig returns required configuration keys
func (p *BasePlugin) GetRequiredConfig() []string {
	return []string{
		"ai.model",
		"ai.api_key",
	}
}

// GetDefaultConfig returns empty default config
func (p *BasePlugin) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{}
}

// LoadPrompt loads a prompt template from the plugin's prompts directory
func LoadPrompt(filename string) (string, error) {
	// This will be implemented by the plugin loader
	// to handle prompt file resolution
	return "", nil
}

// RenderPrompt replaces template variables in a prompt
func RenderPrompt(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}