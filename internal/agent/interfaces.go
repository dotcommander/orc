package agent

import "context"

type AIClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
	CompleteJSON(ctx context.Context, prompt string) (string, error)
	// Enhanced methods with system prompt support
	CompleteWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	CompleteJSONWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}