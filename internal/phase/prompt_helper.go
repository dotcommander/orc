package phase

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

// LoadAndExecutePrompt loads a prompt template file and executes it with the given data
func LoadAndExecutePrompt(promptPath string, data interface{}) (string, error) {
	// Load the prompt template
	promptContent, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("reading prompt file %s: %w", promptPath, err)
	}
	
	// Parse as template
	tmpl, err := template.New("prompt").Parse(string(promptContent))
	if err != nil {
		return "", fmt.Errorf("parsing prompt template: %w", err)
	}
	
	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing prompt template: %w", err)
	}
	
	return buf.String(), nil
}