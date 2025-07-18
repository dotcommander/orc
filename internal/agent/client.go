package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	
	"golang.org/x/time/rate"
)

type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	maxRetries int
	limiter    *rate.Limiter
	apiType    string // "anthropic" or "openai"
	logger     *slog.Logger
}

type Option func(*Client)

func WithRetry(maxRetries int) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		// Preserve existing transport if any
		transport := c.httpClient.Transport
		c.httpClient = &http.Client{
			Timeout:   timeout,
			Transport: transport,
		}
	}
}

func WithRateLimit(requestsPerMinute int, burst int) Option {
	return func(c *Client) {
		c.limiter = rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60.0), burst)
	}
}

func WithAPIConfig(baseURL, model string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
		c.model = model
		// Detect API type based on base URL
		if contains(baseURL, "openai") {
			c.apiType = "openai"
		} else {
			c.apiType = "anthropic"
		}
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func NewClient(apiKey string, opts ...Option) *Client {
	// Configure transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	
	c := &Client{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		model:   "claude-3-5-sonnet-20241022",
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		maxRetries: 3,
		limiter:    rate.NewLimiter(rate.Limit(1), 1), // Default: 60 req/min
		apiType:    "anthropic",
		logger:     slog.Default().With("component", "ai_client"),
	}
	
	for _, opt := range opts {
		opt(c)
	}
	
	c.logger.Debug("AI client initialized",
		"api_type", c.apiType,
		"base_url", c.baseURL,
		"model", c.model,
		"max_retries", c.maxRetries,
		"rate_limit", fmt.Sprintf("%v req/s", c.limiter.Limit()))
	
	return c
}

func (c *Client) Execute(ctx context.Context, prompt string, input any) (string, error) {
	fullPrompt := prompt
	if input != nil {
		if str, ok := input.(string); ok && str != "" {
			fullPrompt = fmt.Sprintf("%s\n\n%s", prompt, str)
		}
	}
	
	return c.Complete(ctx, fullPrompt)
}

func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	return c.complete(ctx, prompt, false)
}

func (c *Client) CompleteJSON(ctx context.Context, prompt string) (string, error) {
	return c.complete(ctx, prompt, true)
}

// CompleteWithSystem makes a request with separate system and user prompts
func (c *Client) CompleteWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.completeWithSystem(ctx, systemPrompt, userPrompt, false)
}

// CompleteJSONWithSystem makes a JSON request with separate system and user prompts
func (c *Client) CompleteJSONWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.completeWithSystem(ctx, systemPrompt, userPrompt, true)
}

func (c *Client) complete(ctx context.Context, prompt string, forceJSON bool) (string, error) {
	requestID := fmt.Sprintf("api_%d", time.Now().UnixNano())
	startTime := time.Now()
	
	c.logger.Debug("waiting for rate limit",
		"request_id", requestID)
	
	if err := c.limiter.Wait(ctx); err != nil {
		c.logger.Error("rate limit wait failed",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("rate limit wait failed: %w", err)
	}
	
	c.logger.Debug("rate limit passed for AI request",
		"request_id", requestID,
		"wait_duration_ms", time.Since(startTime).Milliseconds(),
		"limit_per_second", c.limiter.Limit(),
		"burst_capacity", c.limiter.Burst())
	
	var lastErr error
	
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			c.logger.Debug("retry backoff",
				"request_id", requestID,
				"attempt", attempt,
				"backoff_seconds", backoff.Seconds())
			
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				c.logger.Warn("request cancelled during backoff",
					"request_id", requestID,
					"attempt", attempt)
				return "", ctx.Err()
			}
		}
		
		attemptStart := time.Now()
		// Extract operation type from prompt for better logging
		operationType := extractOperationType(prompt)
		c.logger.Debug("attempting AI generation request",
			"request_id", requestID,
			"attempt", attempt,
			"operation", operationType,
			"prompt_length", len(prompt),
			"force_json", forceJSON,
			"api_type", c.apiType,
			"model", c.model)
		
		response, err := c.doRequest(ctx, prompt, forceJSON)
		attemptDuration := time.Since(attemptStart)
		
		if err == nil {
			c.logger.Info("API request successful",
				"request_id", requestID,
				"attempt", attempt,
				"duration_ms", attemptDuration.Milliseconds(),
				"response_length", len(response),
				"total_duration_ms", time.Since(startTime).Milliseconds())
			return response, nil
		}
		
		lastErr = err
		
		if !isRetryable(err) {
			c.logger.Error("API request failed with non-retryable error",
				"request_id", requestID,
				"attempt", attempt,
				"duration_ms", attemptDuration.Milliseconds(),
				"error", err)
			return "", err
		}
		
		c.logger.Warn("API request failed, will retry",
			"request_id", requestID,
			"attempt", attempt,
			"duration_ms", attemptDuration.Milliseconds(),
			"error", err)
	}
	
	c.logger.Error("API request failed after max retries",
		"request_id", requestID,
		"max_retries", c.maxRetries,
		"total_duration_ms", time.Since(startTime).Milliseconds(),
		"last_error", lastErr)
	
	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) doRequest(ctx context.Context, prompt string, forceJSON bool) (string, error) {
	if c.apiType == "openai" {
		return c.doOpenAIRequest(ctx, prompt, forceJSON)
	}
	return c.doAnthropicRequest(ctx, prompt, forceJSON)
}

func (c *Client) doOpenAIRequest(ctx context.Context, prompt string, forceJSON bool) (string, error) {
	requestID := fmt.Sprintf("openai_%d", time.Now().UnixNano())
	
	operationType := extractOperationType(prompt)
	c.logger.Debug("preparing OpenAI API request",
		"request_id", requestID,
		"operation", operationType,
		"model", c.model,
		"force_json", forceJSON,
		"message_count", 1)
	
	messages := []map[string]string{
		{
			"role":    "user",
			"content": prompt,
		},
	}
	
	// Add system message for JSON mode
	if forceJSON {
		messages = append([]map[string]string{
			{
				"role":    "system",
				"content": "You are a helpful assistant that MUST respond with valid JSON only. Your entire response must be a single JSON object with no additional text, markdown, or explanations.",
			},
		}, messages...)
	}
	
	requestBody := map[string]interface{}{
		"model": c.model,
		"messages": messages,
		"max_tokens": 4096,
	}
	
	if forceJSON {
		requestBody["response_format"] = map[string]string{
			"type": "json_object",
		}
	}
	
	body, err := json.Marshal(requestBody)
	if err != nil {
		c.logger.Error("failed to marshal OpenAI request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("marshaling request: %w", err)
	}
	
	c.logger.Debug("OpenAI request body prepared",
		"request_id", requestID,
		"operation", operationType,
		"body_size_bytes", len(body),
		"max_tokens", 4096,
		"has_json_mode", forceJSON)
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		c.logger.Error("failed to create OpenAI request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	httpStart := time.Now()
	c.logger.Debug("sending OpenAI HTTP request",
		"request_id", requestID,
		"operation", operationType,
		"endpoint", "/chat/completions",
		"method", "POST")
	
	resp, err := c.httpClient.Do(req)
	httpDuration := time.Since(httpStart)
	
	if err != nil {
		c.logger.Error("OpenAI HTTP request failed",
			"request_id", requestID,
			"duration_ms", httpDuration.Milliseconds(),
			"error", err)
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	
	c.logger.Debug("OpenAI HTTP response received",
		"request_id", requestID,
		"status_code", resp.StatusCode,
		"duration_ms", httpDuration.Milliseconds())
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read OpenAI response body",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("reading response: %w", err)
	}
	
	c.logger.Debug("OpenAI response body read",
		"request_id", requestID,
		"body_size", len(respBody))
	
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("OpenAI API error",
			"request_id", requestID,
			"status_code", resp.StatusCode,
			"response_body", string(respBody))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(respBody, &response); err != nil {
		c.logger.Error("failed to parse OpenAI response",
			"request_id", requestID,
			"error", err,
			"response_body", string(respBody))
		return "", fmt.Errorf("parsing response: %w", err)
	}
	
	if len(response.Choices) == 0 {
		c.logger.Error("no choices in OpenAI response",
			"request_id", requestID)
		return "", fmt.Errorf("no choices in response")
	}
	
	c.logger.Info("OpenAI request completed",
		"request_id", requestID,
		"prompt_tokens", response.Usage.PromptTokens,
		"completion_tokens", response.Usage.CompletionTokens,
		"total_tokens", response.Usage.TotalTokens,
		"response_length", len(response.Choices[0].Message.Content))
	
	return response.Choices[0].Message.Content, nil
}

func (c *Client) doAnthropicRequest(ctx context.Context, prompt string, forceJSON bool) (string, error) {
	requestID := fmt.Sprintf("anthropic_%d", time.Now().UnixNano())
	
	operationType := extractOperationType(prompt)
	c.logger.Debug("preparing Anthropic API request",
		"request_id", requestID,
		"operation", operationType,
		"model", c.model,
		"force_json", forceJSON,
		"message_count", 1)
	
	requestBody := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 4096,
	}
	
	if forceJSON {
		requestBody["system"] = "You are a helpful assistant that responds with valid JSON only. Do not include markdown formatting, explanations, or any text outside of the JSON object."
	}
	
	body, err := json.Marshal(requestBody)
	if err != nil {
		c.logger.Error("failed to marshal Anthropic request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("marshaling request: %w", err)
	}
	
	c.logger.Debug("Anthropic request body prepared",
		"request_id", requestID,
		"operation", operationType,
		"body_size_bytes", len(body),
		"max_tokens", 4096,
		"has_system_prompt", forceJSON)
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		c.logger.Error("failed to create Anthropic request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	
	httpStart := time.Now()
	c.logger.Debug("sending Anthropic HTTP request",
		"request_id", requestID,
		"operation", operationType,
		"endpoint", "/messages",
		"method", "POST")
	
	resp, err := c.httpClient.Do(req)
	httpDuration := time.Since(httpStart)
	
	if err != nil {
		c.logger.Error("Anthropic HTTP request failed",
			"request_id", requestID,
			"duration_ms", httpDuration.Milliseconds(),
			"error", err)
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	
	c.logger.Debug("Anthropic HTTP response received",
		"request_id", requestID,
		"status_code", resp.StatusCode,
		"duration_ms", httpDuration.Milliseconds())
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Anthropic response body",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("reading response: %w", err)
	}
	
	c.logger.Debug("Anthropic response body read",
		"request_id", requestID,
		"body_size", len(respBody))
	
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Anthropic API error",
			"request_id", requestID,
			"status_code", resp.StatusCode,
			"response_body", string(respBody))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(respBody, &response); err != nil {
		c.logger.Error("failed to parse Anthropic response",
			"request_id", requestID,
			"error", err,
			"response_body", string(respBody))
		return "", fmt.Errorf("parsing response: %w", err)
	}
	
	if len(response.Content) == 0 {
		c.logger.Error("no content in Anthropic response",
			"request_id", requestID)
		return "", fmt.Errorf("no content in response")
	}
	
	c.logger.Info("Anthropic request completed",
		"request_id", requestID,
		"input_tokens", response.Usage.InputTokens,
		"output_tokens", response.Usage.OutputTokens,
		"total_tokens", response.Usage.InputTokens+response.Usage.OutputTokens,
		"response_length", len(response.Content[0].Text))
	
	return response.Content[0].Text, nil
}

func isRetryable(err error) bool {
	return true
}

// extractOperationType analyzes the prompt to determine what operation is being performed
func extractOperationType(prompt string) string {
	if len(prompt) < 50 {
		return "unknown"
	}
	
	promptLower := strings.ToLower(prompt)
	
	// Novel generation operations
	if strings.Contains(promptLower, "systematic planning") || strings.Contains(promptLower, "novel plan") {
		return "systematic_planning"
	}
	if strings.Contains(promptLower, "write this specific scene") || strings.Contains(promptLower, "scene objective") {
		return "scene_writing"
	}
	if strings.Contains(promptLower, "contextual editing") || strings.Contains(promptLower, "edit this chapter") {
		return "contextual_editing"
	}
	if strings.Contains(promptLower, "systematic assembly") || strings.Contains(promptLower, "assemble the novel") {
		return "systematic_assembly"
	}
	
	// Story development operations
	if strings.Contains(promptLower, "story premise") || strings.Contains(promptLower, "develop this into") {
		return "premise_development"
	}
	if strings.Contains(promptLower, "create characters") || strings.Contains(promptLower, "main characters") {
		return "character_creation"
	}
	if strings.Contains(promptLower, "plot arc") || strings.Contains(promptLower, "story structure") {
		return "plot_development"
	}
	if strings.Contains(promptLower, "chapter") && strings.Contains(promptLower, "scenes") {
		return "chapter_planning"
	}
	
	// Content operations
	if strings.Contains(promptLower, "expand this scene") {
		return "scene_expansion"
	}
	if strings.Contains(promptLower, "tighten this scene") {
		return "scene_tightening"
	}
	if strings.Contains(promptLower, "title") && strings.Contains(promptLower, "story") {
		return "title_generation"
	}
	
	// Fallback classification
	if strings.Contains(promptLower, "write") {
		return "content_writing"
	}
	if strings.Contains(promptLower, "create") || strings.Contains(promptLower, "generate") {
		return "content_generation"
	}
	if strings.Contains(promptLower, "analyze") || strings.Contains(promptLower, "review") {
		return "content_analysis"
	}
	
	return "general_request"
}

// completeWithSystem handles requests with separate system and user prompts
func (c *Client) completeWithSystem(ctx context.Context, systemPrompt, userPrompt string, forceJSON bool) (string, error) {
	requestID := fmt.Sprintf("api_%d", time.Now().UnixNano())
	startTime := time.Now()
	
	c.logger.Debug("waiting for rate limit",
		"request_id", requestID)
	
	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiting: %w", err)
	}
	
	c.logger.Debug("rate limit passed for AI request",
		"request_id", requestID,
		"wait_duration_ms", time.Since(startTime).Milliseconds(),
		"limit_per_second", c.limiter.Limit(),
		"burst_capacity", c.limiter.Burst())
	
	operationType := extractOperationType(userPrompt)
	
	c.logger.Debug("attempting AI generation request",
		"request_id", requestID,
		"attempt", 0,
		"operation", operationType,
		"system_prompt_length", len(systemPrompt),
		"user_prompt_length", len(userPrompt),
		"force_json", forceJSON,
		"api_type", c.apiType,
		"model", c.model)
	
	var response string
	var err error
	
	if c.apiType == "openai" {
		response, err = c.doOpenAIRequestWithSystem(ctx, systemPrompt, userPrompt, forceJSON)
	} else {
		response, err = c.doAnthropicRequestWithSystem(ctx, systemPrompt, userPrompt, forceJSON)
	}
	
	if err != nil {
		c.logger.Error("AI generation request failed",
			"request_id", requestID,
			"operation", operationType,
			"error", err)
		return "", fmt.Errorf("AI generation failed: %w", err)
	}
	
	c.logger.Info("API request successful",
		"request_id", requestID,
		"attempt", 0,
		"duration_ms", time.Since(startTime).Milliseconds(),
		"response_length", len(response),
		"total_duration_ms", time.Since(startTime).Milliseconds())
	
	return response, nil
}

// doOpenAIRequestWithSystem makes OpenAI API request with separate system and user prompts
func (c *Client) doOpenAIRequestWithSystem(ctx context.Context, systemPrompt, userPrompt string, forceJSON bool) (string, error) {
	requestID := fmt.Sprintf("openai_%d", time.Now().UnixNano())
	
	operationType := extractOperationType(userPrompt)
	c.logger.Debug("preparing OpenAI API request with system prompt",
		"request_id", requestID,
		"operation", operationType,
		"model", c.model,
		"force_json", forceJSON,
		"message_count", 2)
	
	messages := []map[string]string{
		{
			"role":    "system",
			"content": systemPrompt,
		},
		{
			"role":    "user",
			"content": userPrompt,
		},
	}
	
	// For JSON mode, enhance system prompt
	if forceJSON {
		messages[0]["content"] = systemPrompt + "\n\nIMPORTANT: You MUST respond with valid JSON only. Your entire response must be a single JSON object with no additional text, markdown, or explanations."
	}
	
	requestBody := map[string]interface{}{
		"model": c.model,
		"messages": messages,
		"max_tokens": 4096,
	}
	
	if forceJSON {
		requestBody["response_format"] = map[string]string{
			"type": "json_object",
		}
	}
	
	body, err := json.Marshal(requestBody)
	if err != nil {
		c.logger.Error("failed to marshal OpenAI request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("marshaling request: %w", err)
	}
	
	c.logger.Debug("OpenAI request body prepared",
		"request_id", requestID,
		"operation", operationType,
		"body_size_bytes", len(body),
		"max_tokens", 4096,
		"has_json_mode", forceJSON)
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		c.logger.Error("failed to create OpenAI request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	httpStart := time.Now()
	c.logger.Debug("sending OpenAI HTTP request",
		"request_id", requestID,
		"operation", operationType,
		"endpoint", "/chat/completions",
		"method", "POST")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("OpenAI HTTP request failed",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	c.logger.Debug("OpenAI HTTP response received",
		"request_id", requestID,
		"status_code", resp.StatusCode,
		"duration_ms", time.Since(httpStart).Milliseconds())
	
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read OpenAI response body",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("reading response: %w", err)
	}
	
	c.logger.Debug("OpenAI response body read",
		"request_id", requestID,
		"body_size", len(responseBody))
	
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("OpenAI API error",
			"request_id", requestID,
			"status_code", resp.StatusCode,
			"response", string(responseBody))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(responseBody))
	}
	
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(responseBody, &response); err != nil {
		c.logger.Error("failed to parse OpenAI response",
			"request_id", requestID,
			"error", err,
			"response", string(responseBody))
		return "", fmt.Errorf("parsing response: %w", err)
	}
	
	if len(response.Choices) == 0 {
		c.logger.Error("no choices in OpenAI response",
			"request_id", requestID,
			"response", string(responseBody))
		return "", fmt.Errorf("no choices in response")
	}
	
	content := response.Choices[0].Message.Content
	
	c.logger.Info("OpenAI request completed",
		"request_id", requestID,
		"prompt_tokens", response.Usage.PromptTokens,
		"completion_tokens", response.Usage.CompletionTokens,
		"total_tokens", response.Usage.TotalTokens,
		"response_length", len(content))
	
	return content, nil
}

// doAnthropicRequestWithSystem makes Anthropic API request with separate system and user prompts  
func (c *Client) doAnthropicRequestWithSystem(ctx context.Context, systemPrompt, userPrompt string, forceJSON bool) (string, error) {
	requestID := fmt.Sprintf("anthropic_%d", time.Now().UnixNano())
	
	operationType := extractOperationType(userPrompt)
	c.logger.Debug("preparing Anthropic API request with system prompt",
		"request_id", requestID,
		"operation", operationType,
		"model", c.model,
		"force_json", forceJSON,
		"message_count", 1)
	
	// Anthropic uses system parameter separate from messages
	finalSystemPrompt := systemPrompt
	if forceJSON {
		finalSystemPrompt = systemPrompt + "\n\nIMPORTANT: You MUST respond with valid JSON only. Your entire response must be a single JSON object with no additional text, markdown, or explanations."
	}
	
	requestBody := map[string]interface{}{
		"model": c.model,
		"system": finalSystemPrompt,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"max_tokens": 4096,
	}
	
	body, err := json.Marshal(requestBody)
	if err != nil {
		c.logger.Error("failed to marshal Anthropic request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("marshaling request: %w", err)
	}
	
	c.logger.Debug("Anthropic request body prepared",
		"request_id", requestID,
		"operation", operationType,
		"body_size_bytes", len(body),
		"max_tokens", 4096,
		"has_system_prompt", true)
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		c.logger.Error("failed to create Anthropic request",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	
	httpStart := time.Now()
	c.logger.Debug("sending Anthropic HTTP request",
		"request_id", requestID,
		"operation", operationType,
		"endpoint", "/messages",
		"method", "POST")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Anthropic HTTP request failed",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	c.logger.Debug("Anthropic HTTP response received",
		"request_id", requestID,
		"status_code", resp.StatusCode,
		"duration_ms", time.Since(httpStart).Milliseconds())
	
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Anthropic response body",
			"request_id", requestID,
			"error", err)
		return "", fmt.Errorf("reading response: %w", err)
	}
	
	c.logger.Debug("Anthropic response body read",
		"request_id", requestID,
		"body_size", len(responseBody))
	
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Anthropic API error",
			"request_id", requestID,
			"status_code", resp.StatusCode,
			"response", string(responseBody))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(responseBody))
	}
	
	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(responseBody, &response); err != nil {
		c.logger.Error("failed to parse Anthropic response",
			"request_id", requestID,
			"error", err,
			"response", string(responseBody))
		return "", fmt.Errorf("parsing response: %w", err)
	}
	
	if len(response.Content) == 0 {
		c.logger.Error("no content in Anthropic response",
			"request_id", requestID,
			"response", string(responseBody))
		return "", fmt.Errorf("no content in response")
	}
	
	content := response.Content[0].Text
	
	c.logger.Info("Anthropic request completed",
		"request_id", requestID,
		"input_tokens", response.Usage.InputTokens,
		"output_tokens", response.Usage.OutputTokens,
		"total_tokens", response.Usage.InputTokens + response.Usage.OutputTokens,
		"response_length", len(content))
	
	return content, nil
}