package agent

import (
	"fmt"
	"os"
	"sync"
	"text/template"
)

// PromptCache caches parsed prompt templates to avoid repeated file reads
type PromptCache struct {
	mu        sync.RWMutex
	templates map[string]*template.Template
	raw       map[string]string
}

// NewPromptCache creates a new prompt cache
func NewPromptCache() *PromptCache {
	return &PromptCache{
		templates: make(map[string]*template.Template),
		raw:       make(map[string]string),
	}
}

// LoadPrompt loads a prompt from file or cache
func (pc *PromptCache) LoadPrompt(path string) (string, error) {
	// Check cache first
	pc.mu.RLock()
	if content, ok := pc.raw[path]; ok {
		pc.mu.RUnlock()
		return content, nil
	}
	pc.mu.RUnlock()
	
	// Load from file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading prompt file: %w", err)
	}
	
	// Cache the content
	pc.mu.Lock()
	pc.raw[path] = string(content)
	pc.mu.Unlock()
	
	return string(content), nil
}

// LoadTemplate loads and parses a template from file or cache
func (pc *PromptCache) LoadTemplate(name, path string) (*template.Template, error) {
	// Check cache first
	pc.mu.RLock()
	if tmpl, ok := pc.templates[path]; ok {
		pc.mu.RUnlock()
		return tmpl, nil
	}
	pc.mu.RUnlock()
	
	// Load content
	content, err := pc.LoadPrompt(path)
	if err != nil {
		return nil, err
	}
	
	// Parse template
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}
	
	// Cache the parsed template
	pc.mu.Lock()
	pc.templates[path] = tmpl
	pc.mu.Unlock()
	
	return tmpl, nil
}

// Clear removes all cached prompts and templates
func (pc *PromptCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	pc.templates = make(map[string]*template.Template)
	pc.raw = make(map[string]string)
}

// Preload loads multiple prompts into cache
func (pc *PromptCache) Preload(paths []string) error {
	for _, path := range paths {
		if _, err := pc.LoadPrompt(path); err != nil {
			return fmt.Errorf("preloading %s: %w", path, err)
		}
	}
	return nil
}

// Stats returns cache statistics
func (pc *PromptCache) Stats() (templates int, raw int) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	return len(pc.templates), len(pc.raw)
}