package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Limits struct {
	MaxConcurrentWriters int               `yaml:"max_concurrent_writers" validate:"required,min=1,max=100"`
	MaxPromptSize        int               `yaml:"max_prompt_size" validate:"required,min=1000,max=1000000"`
	MaxRetries          int               `yaml:"max_retries" validate:"required,min=0,max=10"`
	TotalTimeout        time.Duration     `yaml:"total_timeout" validate:"required,min=1m,max=24h"`
	PhaseTimeouts       PhaseTimeouts     `yaml:"phase_timeouts"`
	RateLimit           RateLimitConfig   `yaml:"rate_limit" validate:"required"`
}

type PhaseTimeouts struct {
	Planning     time.Duration `yaml:"planning" validate:"min=1m,max=6h"`
	Architecture time.Duration `yaml:"architecture" validate:"min=1m,max=6h"`
	Writing      time.Duration `yaml:"writing" validate:"min=5m,max=6h"`
	Assembly     time.Duration `yaml:"assembly" validate:"min=1m,max=6h"`
	Critique     time.Duration `yaml:"critique" validate:"min=1m,max=6h"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute" validate:"required,min=1,max=1000"`
	BurstSize        int `yaml:"burst_size" validate:"required,min=1,max=100"`
}

func DefaultLimits() Limits {
	return Limits{
		MaxConcurrentWriters: 10,
		MaxPromptSize:       200000,
		MaxRetries:         5,
		TotalTimeout:       6 * time.Hour, // Extended from 2 hours to 6 hours
		PhaseTimeouts: PhaseTimeouts{
			Planning:     45 * time.Minute, // Extended from 10 to 45 minutes
			Architecture: 60 * time.Minute, // Extended from 15 to 60 minutes  
			Writing:      3 * time.Hour,    // Extended from 60 minutes to 3 hours
			Assembly:     30 * time.Minute, // Extended from 5 to 30 minutes
			Critique:     45 * time.Minute, // Extended from 10 to 45 minutes
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 30,
			BurstSize:        15,
		},
	}
}

// DefaultPluginsConfig returns default plugin configuration
func DefaultPluginsConfig() PluginsConfig {
	// Get XDG-compliant paths
	var pluginPaths []string
	var builtinPath, externalPath string
	
	// XDG data directory
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		builtinPath = filepath.Join(xdgData, "orchestrator", "plugins", "builtin")
		externalPath = filepath.Join(xdgData, "orchestrator", "plugins", "external")
	} else {
		home, _ := os.UserHomeDir()
		builtinPath = filepath.Join(home, ".local", "share", "orchestrator", "plugins", "builtin")
		externalPath = filepath.Join(home, ".local", "share", "orchestrator", "plugins", "external")
	}
	
	// Standard plugin discovery paths
	pluginPaths = []string{
		builtinPath,
		externalPath,
		"/usr/local/lib/orchestrator/plugins",
		"/usr/lib/orchestrator/plugins",
	}
	
	// Add user's PATH for system-wide plugins
	if binPath := os.Getenv("PATH"); binPath != "" {
		// Add common binary paths for plugins
		pathDirs := filepath.SplitList(binPath)
		for _, dir := range pathDirs {
			if strings.Contains(dir, "bin") {
				pluginPaths = append(pluginPaths, dir)
			}
		}
	}
	
	return PluginsConfig{
		DiscoveryPaths: pluginPaths,
		BuiltinPath:    builtinPath,
		ExternalPath:   externalPath,
		Configurations: make(map[string]PluginConfiguration),
		Settings: PluginSettings{
			AutoDiscovery:      true,
			MaxExternalPlugins: 10,
			LoadTimeout:        "30s",
			EnableSandboxing:   false, // Future enhancement
		},
	}
}