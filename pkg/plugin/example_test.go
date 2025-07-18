package plugin_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dotcommander/orc/internal/domain/plugin"
	pkgPlugin "github.com/dotcommander/orc/pkg/plugin"
)

func ExampleDiscoverer() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create a discoverer
	discoverer := pkgPlugin.NewDiscoverer(logger)

	// Add custom search paths if needed
	discoverer.AddSearchPath("/opt/orchestrator/plugins")

	// Discover all plugins
	manifests, err := discoverer.Discover()
	if err != nil {
		logger.Error("failed to discover plugins", "error", err)
		return
	}

	// List discovered plugins
	for _, manifest := range manifests {
		fmt.Printf("Found plugin: %s v%s (%s)\n", 
			manifest.Name, manifest.Version, manifest.Type)
		fmt.Printf("  Description: %s\n", manifest.Description)
		fmt.Printf("  Domains: %v\n", manifest.Domains)
		fmt.Printf("  Phases: %d\n", len(manifest.Phases))
	}

	// Find plugins for a specific domain
	fictionPlugins, _ := discoverer.DiscoverByDomain("fiction")
	fmt.Printf("\nFiction plugins: %d\n", len(fictionPlugins))

	// Get a specific plugin
	codePlugin, err := discoverer.GetPlugin("code")
	if err == nil {
		fmt.Printf("\nCode plugin location: %s\n", codePlugin.Location)
	}
}

func ExampleLoader() {
	// Create dependencies
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	discoverer := pkgPlugin.NewDiscoverer(logger)
	registry := plugin.NewDomainRegistry()

	// Create loader
	loader := pkgPlugin.NewLoader(logger, discoverer, registry)

	// Load all plugins
	if err := loader.LoadAll(); err != nil {
		logger.Error("failed to load plugins", "error", err)
		return
	}

	// Check loaded plugins
	loaded := loader.GetLoaded()
	fmt.Printf("Loaded %d plugins\n", len(loaded))

	// Check if specific plugin is loaded
	if loader.IsLoaded("fiction") {
		fmt.Println("Fiction plugin is loaded")
	}

	// Reload a plugin
	if err := loader.Reload("code"); err != nil {
		logger.Error("failed to reload plugin", "error", err)
	}
}

func ExampleManifest() {
	// Create a new manifest programmatically
	manifest := &pkgPlugin.Manifest{
		Name:        "example-plugin",
		Version:     "1.0.0",
		Description: "An example plugin",
		Author:      "Example Author",
		License:     "MIT",
		Type:        pkgPlugin.PluginTypeExternal,
		Domains:     []string{"fiction", "docs"},
		Created:     time.Now(),
		Updated:     time.Now(),
		Phases: []pkgPlugin.PhaseDefinition{
			{
				Name:          "analyze",
				Description:   "Analyze the input",
				Order:         1,
				Required:      true,
				EstimatedTime: 5 * time.Minute,
				Timeout:       10 * time.Minute,
				Retryable:     true,
				MaxRetries:    3,
			},
			{
				Name:          "generate",
				Description:   "Generate output",
				Order:         2,
				Required:      true,
				EstimatedTime: 10 * time.Minute,
				Timeout:       20 * time.Minute,
				Retryable:     true,
				MaxRetries:    2,
			},
		},
		Prompts: map[string]string{
			"analyze":  "prompts/analyze.txt",
			"generate": "prompts/generate.txt",
		},
		OutputSpec: pkgPlugin.OutputSpec{
			PrimaryOutput:    "output.md",
			SecondaryOutputs: []string{"metadata.json"},
			Descriptions: map[string]string{
				"output.md":     "Main output file",
				"metadata.json": "Metadata about the generation",
			},
		},
		DefaultConfig: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2000,
		},
	}

	// Save manifest to file
	tempDir := os.TempDir()
	manifestPath := filepath.Join(tempDir, "example-plugin", "plugin.yaml")
	
	if err := pkgPlugin.SaveManifest(manifest, manifestPath); err != nil {
		fmt.Printf("Error saving manifest: %v\n", err)
		return
	}

	fmt.Printf("Manifest saved to: %s\n", manifestPath)

	// Load manifest from file
	loaded, err := pkgPlugin.LoadManifest(manifestPath)
	if err != nil {
		fmt.Printf("Error loading manifest: %v\n", err)
		return
	}

	fmt.Printf("Loaded plugin: %s\n", loaded.String())
}

func TestPluginCompatibility(t *testing.T) {
	manifest := &pkgPlugin.Manifest{
		Name:       "test-plugin",
		Version:    "1.0.0",
		MinVersion: "1.0.0",
		MaxVersion: "2.0.0",
	}

	// Test compatibility (currently always returns true)
	if !manifest.IsCompatible("1.5.0") {
		t.Error("Expected plugin to be compatible with version 1.5.0")
	}
}

func TestManifestValidation(t *testing.T) {
	tests := []struct {
		name    string
		manifest *pkgPlugin.Manifest
		wantErr bool
	}{
		{
			name: "valid manifest",
			manifest: &pkgPlugin.Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    pkgPlugin.PluginTypeBuiltin,
				Domains: []string{"fiction"},
				Phases: []pkgPlugin.PhaseDefinition{
					{Name: "phase1"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			manifest: &pkgPlugin.Manifest{
				Version: "1.0.0",
				Type:    pkgPlugin.PluginTypeBuiltin,
				Domains: []string{"fiction"},
				Phases: []pkgPlugin.PhaseDefinition{
					{Name: "phase1"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid domain",
			manifest: &pkgPlugin.Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    pkgPlugin.PluginTypeBuiltin,
				Domains: []string{"invalid-domain"},
				Phases: []pkgPlugin.PhaseDefinition{
					{Name: "phase1"},
				},
			},
			wantErr: true,
		},
		{
			name: "no phases",
			manifest: &pkgPlugin.Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    pkgPlugin.PluginTypeBuiltin,
				Domains: []string{"fiction"},
				Phases:  []pkgPlugin.PhaseDefinition{},
			},
			wantErr: true,
		},
		{
			name: "duplicate phase names",
			manifest: &pkgPlugin.Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    pkgPlugin.PluginTypeBuiltin,
				Domains: []string{"fiction"},
				Phases: []pkgPlugin.PhaseDefinition{
					{Name: "phase1"},
					{Name: "phase1"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}