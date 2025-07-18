package config

import (
	"strings"
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				AI: AIConfig{
					APIKey:  "sk-1234567890abcdef1234567890abcdef",
					Model:   "claude-3-5-sonnet-20241022",
					BaseURL: "https://api.anthropic.com/v1",
					Timeout: 30,
				},
				Paths: PathsConfig{
					OutputDir: "output",
					Prompts: PromptsConfig{
						Orchestrator: "prompts/orchestrator.txt",
						Architect:    "prompts/architect.txt",
						Writer:       "prompts/writer.txt",
						Critic:       "prompts/critic.txt",
					},
				},
				Limits: Limits{
					MaxConcurrentWriters: 10,
					MaxPromptSize:        100000,
					MaxRetries:          3,
					TotalTimeout:        30 * time.Minute,
					RateLimit: RateLimitConfig{
						RequestsPerMinute: 60,
						BurstSize:        10,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid API key - too short",
			config: Config{
				AI: AIConfig{
					APIKey:  "short",
					Model:   "claude-3-5-sonnet-20241022",
					BaseURL: "https://api.anthropic.com/v1",
					Timeout: 30,
				},
				Paths: PathsConfig{
					OutputDir: "output",
					Prompts: PromptsConfig{
						Orchestrator: "prompts/orchestrator.txt",
						Architect:    "prompts/architect.txt",
						Writer:       "prompts/writer.txt",
						Critic:       "prompts/critic.txt",
					},
				},
				Limits: DefaultLimits(),
			},
			wantErr: true,
			errMsg:  "APIKey",
		},
		{
			name: "invalid model",
			config: Config{
				AI: AIConfig{
					APIKey:  "sk-1234567890abcdef1234567890abcdef",
					Model:   "invalid-model",
					BaseURL: "https://api.anthropic.com/v1",
					Timeout: 30,
				},
				Paths: PathsConfig{
					OutputDir: "output",
					Prompts: PromptsConfig{
						Orchestrator: "prompts/orchestrator.txt",
						Architect:    "prompts/architect.txt",
						Writer:       "prompts/writer.txt",
						Critic:       "prompts/critic.txt",
					},
				},
				Limits: DefaultLimits(),
			},
			wantErr: true,
			errMsg:  "Model",
		},
		{
			name: "invalid base URL",
			config: Config{
				AI: AIConfig{
					APIKey:  "sk-1234567890abcdef1234567890abcdef",
					Model:   "claude-3-5-sonnet-20241022",
					BaseURL: "not-a-url",
					Timeout: 30,
				},
				Paths: PathsConfig{
					OutputDir: "output",
					Prompts: PromptsConfig{
						Orchestrator: "prompts/orchestrator.txt",
						Architect:    "prompts/architect.txt",
						Writer:       "prompts/writer.txt",
						Critic:       "prompts/critic.txt",
					},
				},
				Limits: DefaultLimits(),
			},
			wantErr: true,
			errMsg:  "BaseURL",
		},
		{
			name: "timeout too high",
			config: Config{
				AI: AIConfig{
					APIKey:  "sk-1234567890abcdef1234567890abcdef",
					Model:   "claude-3-5-sonnet-20241022",
					BaseURL: "https://api.anthropic.com/v1",
					Timeout: 2000,
				},
				Paths: PathsConfig{
					OutputDir: "output",
					Prompts: PromptsConfig{
						Orchestrator: "prompts/orchestrator.txt",
						Architect:    "prompts/architect.txt",
						Writer:       "prompts/writer.txt",
						Critic:       "prompts/critic.txt",
					},
				},
				Limits: DefaultLimits(),
			},
			wantErr: true,
			errMsg:  "Timeout",
		},
		{
			name: "concurrent writers too high",
			config: Config{
				AI: AIConfig{
					APIKey:  "sk-1234567890abcdef1234567890abcdef",
					Model:   "claude-3-5-sonnet-20241022",
					BaseURL: "https://api.anthropic.com/v1",
					Timeout: 30,
				},
				Paths: PathsConfig{
					OutputDir: "output",
					Prompts: PromptsConfig{
						Orchestrator: "prompts/orchestrator.txt",
						Architect:    "prompts/architect.txt",
						Writer:       "prompts/writer.txt",
						Critic:       "prompts/critic.txt",
					},
				},
				Limits: Limits{
					MaxConcurrentWriters: 200,
					MaxPromptSize:        100000,
					MaxRetries:          3,
					TotalTimeout:        30 * time.Minute,
					RateLimit: RateLimitConfig{
						RequestsPerMinute: 60,
						BurstSize:        10,
					},
				},
			},
			wantErr: true,
			errMsg:  "MaxConcurrentWriters",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	
	// Create a config with defaults
	cfg := Config{
		AI: AIConfig{
			APIKey:  "sk-1234567890abcdef1234567890abcdef",
			Model:   "claude-3-5-sonnet-20241022",
			BaseURL: "https://api.anthropic.com/v1",
			Timeout: 30,
		},
		Paths: PathsConfig{
			OutputDir: "output",
			Prompts: PromptsConfig{
				Orchestrator: "prompts/orchestrator.txt",
				Architect:    "prompts/architect.txt",
				Writer:       "prompts/writer.txt",
				Critic:       "prompts/critic.txt",
			},
		},
		Limits: limits,
	}
	
	// Should validate successfully
	if err := cfg.validate(); err != nil {
		t.Errorf("DefaultLimits() should produce valid config, got error: %v", err)
	}
}