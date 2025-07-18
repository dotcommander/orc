package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templates embed.FS

type PluginData struct {
	Name        string
	Domain      string
	Package     string
	Description string
	Author      string
	Version     string
	GitRepo     string
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: orc-plugin create <name> <domain>")
		fmt.Println("Example: orc-plugin create poetry fiction")
		os.Exit(1)
	}

	command := os.Args[1]
	if command != "create" {
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}

	name := os.Args[2]
	domain := os.Args[3]

	// Validate domain
	validDomains := []string{"fiction", "code", "docs", "custom"}
	valid := false
	for _, d := range validDomains {
		if d == domain {
			valid = true
			break
		}
	}
	if !valid {
		fmt.Printf("Invalid domain: %s. Must be one of: %v\n", domain, validDomains)
		os.Exit(1)
	}

	// Create plugin directory
	pluginDir := fmt.Sprintf("orchestrator-%s-plugin", name)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Prepare template data
	data := PluginData{
		Name:        name,
		Domain:      domain,
		Package:     strings.ReplaceAll(name, "-", "_"),
		Description: fmt.Sprintf("Orchestrator plugin for %s generation", name),
		Author:      os.Getenv("USER"),
		Version:     "0.1.0",
		GitRepo:     fmt.Sprintf("github.com/%s/orchestrator-%s-plugin", os.Getenv("USER"), name),
	}

	// Generate files from templates
	files := map[string]string{
		"templates/plugin.go.tmpl":      filepath.Join(pluginDir, "plugin.go"),
		"templates/manifest.yaml.tmpl":  filepath.Join(pluginDir, "manifest.yaml"),
		"templates/go.mod.tmpl":         filepath.Join(pluginDir, "go.mod"),
		"templates/README.md.tmpl":      filepath.Join(pluginDir, "README.md"),
		"templates/Makefile.tmpl":       filepath.Join(pluginDir, "Makefile"),
		"templates/.gitignore.tmpl":     filepath.Join(pluginDir, ".gitignore"),
		"templates/example_test.go.tmpl": filepath.Join(pluginDir, "plugin_test.go"),
	}

	for tmplPath, outPath := range files {
		if err := generateFile(tmplPath, outPath, data); err != nil {
			fmt.Printf("Error generating %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("✅ Created %s\n", outPath)
	}

	// Create prompts directory
	promptsDir := filepath.Join(pluginDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		fmt.Printf("Error creating prompts directory: %v\n", err)
		os.Exit(1)
	}

	// Create example prompt
	promptFile := filepath.Join(promptsDir, "planning.txt")
	promptContent := fmt.Sprintf(`You are a %s creation expert. Your task is to plan a comprehensive %s based on the user's request.

User Request: {{.Request}}

Please provide a detailed plan including:
1. Main themes and concepts
2. Structure and organization
3. Key elements to include
4. Estimated length/scope

Format your response as JSON with the following structure:
{
  "title": "...",
  "summary": "...",
  "structure": [...],
  "key_elements": [...],
  "estimated_length": "..."
}`, name, name)

	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		fmt.Printf("Error creating prompt file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Created %s\n", promptFile)

	fmt.Printf("\n🎉 Plugin scaffold created successfully!\n\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("1. cd %s\n", pluginDir)
	fmt.Printf("2. go mod tidy\n")
	fmt.Printf("3. Edit plugin.go to implement your phases\n")
	fmt.Printf("4. Add prompts to the prompts/ directory\n")
	fmt.Printf("5. make build to compile\n")
	fmt.Printf("6. make test to run tests\n")
	fmt.Printf("7. make install to install locally\n")
}

func generateFile(tmplPath, outPath string, data PluginData) error {
	tmplContent, err := templates.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(tmplContent))
	if err != nil {
		return err
	}

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}