package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AI     AIConfig    `yaml:"ai" validate:"required"`
	Paths  PathsConfig `yaml:"paths" validate:"required"`
	Limits Limits      `yaml:"limits" validate:"required"`
}

type AIConfig struct {
	APIKey   string `yaml:"api_key" validate:"required,min=20"`
	Model    string `yaml:"model" validate:"required,oneof=claude-3-5-sonnet-20241022 claude-3-opus-20240229 gpt-4 gpt-4-turbo gpt-4-turbo-preview gpt-4.1 gpt-4o-mini"`
	BaseURL  string `yaml:"base_url" validate:"required,url"`
	Timeout  int    `yaml:"timeout" validate:"required,min=10,max=3600"`
}

type PathsConfig struct {
	OutputDir string       `yaml:"output_dir" validate:"required,dirpath"`
	Prompts   PromptsConfig `yaml:"prompts" validate:"required"`
}

type PromptsConfig struct {
	Orchestrator string `yaml:"orchestrator" validate:"required,filepath"`
	Architect    string `yaml:"architect" validate:"required,filepath"`
	Writer       string `yaml:"writer" validate:"required,filepath"`
	Critic       string `yaml:"critic" validate:"required,filepath"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	
	configPath := getConfigPath()
	
	// Check if config exists, if not, create it interactively
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		cfg, createErr := createConfigInteractively(configPath)
		if createErr != nil {
			return nil, fmt.Errorf("creating config: %w", createErr)
		}
		return cfg, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	
	// Try to get API key from environment
	if cfg.AI.APIKey == "" || cfg.AI.APIKey == "${OPENAI_API_KEY}" {
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			cfg.AI.APIKey = apiKey
		} else {
			// Prompt for API key if missing
			apiKey, promptErr := promptForAPIKey()
			if promptErr != nil {
				return nil, fmt.Errorf("getting API key: %w", promptErr)
			}
			cfg.AI.APIKey = apiKey
		}
	}
	
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}
	
	return &cfg, nil
}

func getConfigPath() string {
	// 1. Explicit config path via environment variable
	if path := os.Getenv("ORCHESTRATOR_CONFIG"); path != "" {
		return path
	}
	
	// 2. Command line flag takes precedence (handled in main)
	
	// 3. XDG_CONFIG_HOME (XDG Base Directory Specification)
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "orchestrator", "config.yaml")
	}
	
	// 4. Default to ~/.config/orchestrator/config.yaml (XDG fallback)
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "orchestrator", "config.yaml")
}

// expandTilde expands a tilde (~) at the beginning of a path to the user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return original path if we can't get home dir
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func (c *Config) validate() error {
	// Set XDG-compliant defaults before validation
	if c.Paths.OutputDir == "" {
		// Default output to XDG_DATA_HOME/orchestrator or ~/.local/share/orchestrator
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			c.Paths.OutputDir = filepath.Join(xdgData, "orchestrator", "output")
		} else {
			home, _ := os.UserHomeDir()
			c.Paths.OutputDir = filepath.Join(home, ".local", "share", "orchestrator", "output")
		}
	} else {
		// Expand tilde in output directory path
		c.Paths.OutputDir = expandTilde(c.Paths.OutputDir)
	}
	
	// Set default prompt paths to XDG data directory
	dataDir := func() string {
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return filepath.Join(xdgData, "orchestrator", "prompts")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "orchestrator", "prompts")
	}()
	
	if c.Paths.Prompts.Orchestrator == "" {
		c.Paths.Prompts.Orchestrator = filepath.Join(dataDir, "orchestrator.txt")
	} else {
		c.Paths.Prompts.Orchestrator = expandTilde(c.Paths.Prompts.Orchestrator)
	}
	if c.Paths.Prompts.Architect == "" {
		c.Paths.Prompts.Architect = filepath.Join(dataDir, "architect.txt")
	} else {
		c.Paths.Prompts.Architect = expandTilde(c.Paths.Prompts.Architect)
	}
	if c.Paths.Prompts.Writer == "" {
		c.Paths.Prompts.Writer = filepath.Join(dataDir, "writer.txt")
	} else {
		c.Paths.Prompts.Writer = expandTilde(c.Paths.Prompts.Writer)
	}
	if c.Paths.Prompts.Critic == "" {
		c.Paths.Prompts.Critic = filepath.Join(dataDir, "critic.txt")
	} else {
		c.Paths.Prompts.Critic = expandTilde(c.Paths.Prompts.Critic)
	}
	
	if c.Limits.MaxConcurrentWriters == 0 {
		c.Limits = DefaultLimits()
	}
	
	// Use validator for structured validation
	validate := validator.New()
	
	// Register custom validation for dirpath
	validate.RegisterValidation("dirpath", func(fl validator.FieldLevel) bool {
		// For output directory, we'll create it if it doesn't exist
		return true
	})
	
	// Register custom validation for filepath
	validate.RegisterValidation("filepath", func(fl validator.FieldLevel) bool {
		// For prompt files, we just check they're not empty
		// Actual file existence will be checked at runtime
		return fl.Field().String() != ""
	})
	
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	
	return nil
}

// createConfigInteractively creates a new config file with user input
func createConfigInteractively(configPath string) (*Config, error) {
	fmt.Printf("🚀 Welcome to Refiner! Let's set up your configuration.\n\n")
	
	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}
	
	// Get API provider choice
	fmt.Printf("Which AI provider would you like to use?\n")
	fmt.Printf("1. OpenAI (GPT-4, GPT-4-turbo)\n")
	fmt.Printf("2. Anthropic (Claude)\n")
	fmt.Printf("Enter choice (1 or 2): ")
	
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	choice := strings.TrimSpace(scanner.Text())
	
	var cfg Config
	if choice == "1" {
		cfg = createOpenAIConfig()
	} else if choice == "2" {
		cfg = createAnthropicConfig()
	} else {
		// Default to OpenAI
		fmt.Printf("Defaulting to OpenAI...\n")
		cfg = createOpenAIConfig()
	}
	
	// Get API key
	apiKey, err := promptForAPIKey()
	if err != nil {
		return nil, err
	}
	cfg.AI.APIKey = apiKey
	
	// Set up paths
	cfg.setupDefaultPaths()
	
	// Create necessary directories and files
	if err := createDirectoriesAndFiles(&cfg); err != nil {
		return nil, fmt.Errorf("setting up directories: %w", err)
	}
	
	// Save config
	if err := saveConfig(&cfg, configPath); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}
	
	fmt.Printf("\n✅ Configuration saved to: %s\n", configPath)
	fmt.Printf("✅ Ready to generate novels!\n\n")
	
	return &cfg, nil
}

func createOpenAIConfig() Config {
	return Config{
		AI: AIConfig{
			Model:   "gpt-4.1",
			BaseURL: "https://api.openai.com/v1",
			Timeout: 900, // Extended from 300 to 900 seconds (15 minutes)
		},
		Limits: DefaultLimits(),
	}
}

func createAnthropicConfig() Config {
	return Config{
		AI: AIConfig{
			Model:   "claude-3-5-sonnet-20241022",
			BaseURL: "https://api.anthropic.com",
			Timeout: 900, // Extended from 300 to 900 seconds (15 minutes)
		},
		Limits: DefaultLimits(),
	}
}

func promptForAPIKey() (string, error) {
	fmt.Printf("\nPlease enter your API key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	apiKey := strings.TrimSpace(scanner.Text())
	
	if apiKey == "" {
		return "", fmt.Errorf("API key is required")
	}
	
	if len(apiKey) < 20 {
		return "", fmt.Errorf("API key seems too short (minimum 20 characters)")
	}
	
	return apiKey, nil
}

func (c *Config) setupDefaultPaths() {
	// Set XDG-compliant defaults
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		c.Paths.OutputDir = filepath.Join(xdgData, "orchestrator", "output")
	} else {
		home, _ := os.UserHomeDir()
		c.Paths.OutputDir = filepath.Join(home, ".local", "share", "orchestrator", "output")
	}
	
	dataDir := func() string {
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return filepath.Join(xdgData, "orchestrator", "prompts")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "orchestrator", "prompts")
	}()
	
	c.Paths.Prompts = PromptsConfig{
		Orchestrator: filepath.Join(dataDir, "orchestrator.txt"),
		Architect:    filepath.Join(dataDir, "architect.txt"),
		Writer:       filepath.Join(dataDir, "writer.txt"),
		Critic:       filepath.Join(dataDir, "critic.txt"),
	}
}

func createDirectoriesAndFiles(cfg *Config) error {
	// Create output directory
	if err := os.MkdirAll(cfg.Paths.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	
	// Create prompts directory
	promptsDir := filepath.Dir(cfg.Paths.Prompts.Orchestrator)
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return fmt.Errorf("creating prompts directory: %w", err)
	}
	
	// Copy default prompts if they don't exist
	promptFiles := []string{
		cfg.Paths.Prompts.Orchestrator,
		cfg.Paths.Prompts.Architect,
		cfg.Paths.Prompts.Writer,
		cfg.Paths.Prompts.Critic,
	}
	
	for _, promptFile := range promptFiles {
		if _, err := os.Stat(promptFile); os.IsNotExist(err) {
			// Skip creating prompt files if they don't exist - the app will use defaults
			// This removes the dependency on relative paths that break global tool functionality
		}
	}
	
	return nil
}

func saveConfig(cfg *Config, configPath string) error {
	// Use placeholder for API key in saved config for security
	cfgToSave := *cfg
	cfgToSave.AI.APIKey = "${OPENAI_API_KEY}" // Use env var placeholder
	
	data, err := yaml.Marshal(&cfgToSave)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	
	return os.WriteFile(configPath, data, 0644)
}