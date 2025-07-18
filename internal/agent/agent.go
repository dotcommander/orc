package agent

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	globalPromptCache *PromptCache
	cacheOnce         sync.Once
)

// GetPromptCache returns the global prompt cache instance
func GetPromptCache() *PromptCache {
	cacheOnce.Do(func() {
		globalPromptCache = NewPromptCache()
	})
	return globalPromptCache
}

type Agent struct {
	client       AIClient
	promptPath   string
	systemPrompt string // New: System prompt for role assignment
	promptCache  *PromptCache
	logger       *slog.Logger
}

func New(client AIClient, promptPath string) *Agent {
	return &Agent{
		client:      client,
		promptPath:  promptPath,
		promptCache: GetPromptCache(),
		logger:      slog.Default().With("component", "agent"),
	}
}

// NewWithSystem creates an agent with both prompt path and system prompt
func NewWithSystem(client AIClient, promptPath, systemPrompt string) *Agent {
	return &Agent{
		client:       client,
		promptPath:   promptPath,
		systemPrompt: systemPrompt,
		promptCache:  GetPromptCache(),
		logger:       slog.Default().With("component", "agent"),
	}
}

// WithLogger sets a custom logger for the agent
func (a *Agent) WithLogger(logger *slog.Logger) *Agent {
	a.logger = logger.With("component", "agent")
	return a
}

func (a *Agent) Execute(ctx context.Context, prompt string, input any) (string, error) {
	return a.execute(ctx, prompt, input, false)
}

func (a *Agent) ExecuteJSON(ctx context.Context, prompt string, input any) (string, error) {
	return a.execute(ctx, prompt, input, true)
}

func (a *Agent) execute(ctx context.Context, prompt string, input any, forceJSON bool) (string, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	
	// Extract operation type for better logging
	operationType := extractOperationType(prompt)
	a.logger.Debug("starting agent execution",
		"request_id", requestID,
		"operation", operationType,
		"force_json", forceJSON,
		"has_prompt_path", a.promptPath != "",
		"prompt_length", len(prompt))
	
	fullPrompt := prompt
	cacheHit := false
	
	if a.promptPath != "" {
		// Try to load as template first
		tmpl, err := a.promptCache.LoadTemplate("agent", a.promptPath)
		if err == nil {
			a.logger.Debug("loaded prompt template",
				"request_id", requestID,
				"template_path", a.promptPath)
			
			// Create context data combining input and prompt
			templateData := map[string]any{
				"Input":   input,
				"Prompt":  prompt,
				"Context": prompt, // Legacy alias
			}
			
			// If input is a string, also add it as UserRequest
			if str, ok := input.(string); ok {
				templateData["UserRequest"] = str
			}
			
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, templateData); err == nil {
				fullPrompt = buf.String()
				cacheHit = true
				a.logger.Debug("template execution successful",
					"request_id", requestID,
					"template_output_length", len(fullPrompt))
			} else {
				a.logger.Warn("template execution failed, falling back to raw prompt",
					"request_id", requestID,
					"error", err)
				
				// Template execution failed, fall back to raw prompt
				cachedPrompt, loadErr := a.promptCache.LoadPrompt(a.promptPath)
				if loadErr == nil {
					fullPrompt = cachedPrompt + "\n\n" + prompt
					cacheHit = true
					a.logger.Debug("loaded raw prompt from cache",
						"request_id", requestID,
						"cached_prompt_length", len(cachedPrompt))
				} else {
					a.logger.Error("failed to load raw prompt",
						"request_id", requestID,
						"path", a.promptPath,
						"error", loadErr)
				}
			}
		} else {
			// Not a template, try raw prompt
			cachedPrompt, loadErr := a.promptCache.LoadPrompt(a.promptPath)
			if loadErr == nil {
				fullPrompt = cachedPrompt + "\n\n" + prompt
				cacheHit = true
				a.logger.Debug("loaded raw prompt from cache",
					"request_id", requestID,
					"cached_prompt_length", len(cachedPrompt))
			} else {
				a.logger.Error("failed to load prompt",
					"request_id", requestID,
					"path", a.promptPath,
					"error", loadErr)
			}
		}
	}
	
	a.logger.Debug("executing AI request",
		"request_id", requestID,
		"operation", operationType,
		"cache_hit", cacheHit,
		"full_prompt_length", len(fullPrompt),
		"force_json", forceJSON)
	
	var response string
	var err error
	
	// Use system prompt if available
	if a.systemPrompt != "" {
		if forceJSON {
			response, err = a.client.CompleteJSONWithSystem(ctx, a.systemPrompt, fullPrompt)
		} else {
			response, err = a.client.CompleteWithSystem(ctx, a.systemPrompt, fullPrompt)
		}
	} else {
		// Fall back to original behavior
		if forceJSON {
			if client, ok := a.client.(*Client); ok {
				response, err = client.CompleteJSON(ctx, fullPrompt)
			} else {
				response, err = a.client.Complete(ctx, fullPrompt)
			}
		} else {
			response, err = a.client.Complete(ctx, fullPrompt)
		}
	}
	
	duration := time.Since(startTime)
	
	if err != nil {
		a.logger.Error("AI request failed",
			"request_id", requestID,
			"duration_ms", duration.Milliseconds(),
			"error", err)
		return "", err
	}
	
	a.logger.Info("AI request completed",
		"request_id", requestID,
		"operation", operationType,
		"duration_ms", duration.Milliseconds(),
		"response_length", len(response),
		"cache_hit", cacheHit)
	
	return response, nil
}

