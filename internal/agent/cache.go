package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	
	"github.com/dotcommander/orc/internal/storage"
)

type ResponseCache struct {
	storage storage.Storage
	ttl     time.Duration
	logger  *slog.Logger
}

type CachedResponse struct {
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

func NewResponseCache(storage storage.Storage, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		storage: storage,
		ttl:     ttl,
		logger:  slog.Default().With("component", "response_cache"),
	}
}

func (c *ResponseCache) Get(ctx context.Context, prompt string) (string, bool) {
	key := c.hashPrompt(prompt)
	path := fmt.Sprintf("cache/responses/%s.json", key)
	
	c.logger.Debug("cache lookup",
		"key", key,
		"prompt_length", len(prompt))
	
	data, err := c.storage.Load(ctx, path)
	if err != nil {
		c.logger.Debug("cache miss - not found",
			"key", key,
			"error", err)
		return "", false
	}
	
	var cached CachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		c.logger.Error("cache miss - invalid data",
			"key", key,
			"error", err)
		return "", false
	}
	
	age := time.Since(cached.Timestamp)
	if age > c.ttl {
		c.logger.Debug("cache miss - expired",
			"key", key,
			"age", age,
			"ttl", c.ttl)
		return "", false
	}
	
	c.logger.Info("cache hit",
		"key", key,
		"age", age,
		"response_length", len(cached.Response))
	
	return cached.Response, true
}

func (c *ResponseCache) Set(ctx context.Context, prompt, response string) error {
	key := c.hashPrompt(prompt)
	path := fmt.Sprintf("cache/responses/%s.json", key)
	
	c.logger.Debug("cache set",
		"key", key,
		"prompt_length", len(prompt),
		"response_length", len(response))
	
	cached := CachedResponse{
		Response:  response,
		Timestamp: time.Now(),
	}
	
	data, err := json.Marshal(cached)
	if err != nil {
		c.logger.Error("failed to marshal cache entry",
			"key", key,
			"error", err)
		return fmt.Errorf("marshaling cached response: %w", err)
	}
	
	if err := c.storage.Save(ctx, path, data); err != nil {
		c.logger.Error("failed to save cache entry",
			"key", key,
			"error", err)
		return err
	}
	
	c.logger.Info("cache entry saved",
		"key", key,
		"size", len(data))
	
	return nil
}

func (c *ResponseCache) hashPrompt(prompt string) string {
	hash := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(hash[:])
}

type CachedClient struct {
	AIClient
	cache  *ResponseCache
	logger *slog.Logger
}

func WithCache(client AIClient, cache *ResponseCache) AIClient {
	return &CachedClient{
		AIClient: client,
		cache:    cache,
		logger:   slog.Default().With("component", "cached_client"),
	}
}

func (c *CachedClient) Complete(ctx context.Context, prompt string) (string, error) {
	startTime := time.Now()
	
	if response, found := c.cache.Get(ctx, prompt); found {
		c.logger.Info("serving from cache",
			"prompt_length", len(prompt),
			"response_length", len(response),
			"duration_ms", time.Since(startTime).Milliseconds())
		return response, nil
	}
	
	c.logger.Debug("cache miss, calling underlying client",
		"prompt_length", len(prompt))
	
	response, err := c.AIClient.Complete(ctx, prompt)
	if err != nil {
		c.logger.Error("underlying client failed",
			"error", err)
		return "", err
	}
	
	if cacheErr := c.cache.Set(ctx, prompt, response); cacheErr != nil {
		c.logger.Warn("failed to cache response",
			"error", cacheErr)
	}
	
	c.logger.Info("completed with fresh response",
		"prompt_length", len(prompt),
		"response_length", len(response),
		"duration_ms", time.Since(startTime).Milliseconds())
	
	return response, nil
}

func (c *CachedClient) CompleteJSON(ctx context.Context, prompt string) (string, error) {
	startTime := time.Now()
	
	// Create a unique cache key for JSON requests to avoid collisions
	jsonKey := fmt.Sprintf("JSON:%s", prompt)
	
	if response, found := c.cache.Get(ctx, jsonKey); found {
		c.logger.Info("serving JSON from cache",
			"prompt_length", len(prompt),
			"response_length", len(response),
			"duration_ms", time.Since(startTime).Milliseconds())
		return response, nil
	}
	
	c.logger.Debug("cache miss for JSON, calling underlying client",
		"prompt_length", len(prompt))
	
	response, err := c.AIClient.CompleteJSON(ctx, prompt)
	if err != nil {
		c.logger.Error("underlying JSON client failed",
			"error", err)
		return "", err
	}
	
	if cacheErr := c.cache.Set(ctx, jsonKey, response); cacheErr != nil {
		c.logger.Warn("failed to cache JSON response",
			"error", cacheErr)
	}
	
	c.logger.Info("completed JSON with fresh response",
		"prompt_length", len(prompt),
		"response_length", len(response),
		"duration_ms", time.Since(startTime).Milliseconds())
	
	return response, nil
}

// CompleteWithSystem makes a request with separate system and user prompts, with caching
func (c *CachedClient) CompleteWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	startTime := time.Now()
	
	// Create cache key combining system and user prompts
	cacheKey := fmt.Sprintf("SYSTEM:%s|USER:%s", systemPrompt, userPrompt)
	
	if response, found := c.cache.Get(ctx, cacheKey); found {
		c.logger.Info("serving system prompt response from cache",
			"system_prompt_length", len(systemPrompt),
			"user_prompt_length", len(userPrompt),
			"response_length", len(response),
			"duration_ms", time.Since(startTime).Milliseconds())
		return response, nil
	}
	
	c.logger.Debug("cache miss for system prompt, calling underlying client",
		"system_prompt_length", len(systemPrompt),
		"user_prompt_length", len(userPrompt))
	
	response, err := c.AIClient.CompleteWithSystem(ctx, systemPrompt, userPrompt)
	if err != nil {
		c.logger.Error("underlying client failed for system prompt",
			"error", err)
		return "", err
	}
	
	if cacheErr := c.cache.Set(ctx, cacheKey, response); cacheErr != nil {
		c.logger.Warn("failed to cache system prompt response",
			"error", cacheErr)
	}
	
	c.logger.Info("completed system prompt with fresh response",
		"system_prompt_length", len(systemPrompt),
		"user_prompt_length", len(userPrompt),
		"response_length", len(response),
		"duration_ms", time.Since(startTime).Milliseconds())
	
	return response, nil
}

// CompleteJSONWithSystem makes a JSON request with separate system and user prompts, with caching
func (c *CachedClient) CompleteJSONWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	startTime := time.Now()
	
	// Create cache key for JSON system requests
	cacheKey := fmt.Sprintf("JSON_SYSTEM:%s|USER:%s", systemPrompt, userPrompt)
	
	if response, found := c.cache.Get(ctx, cacheKey); found {
		c.logger.Info("serving JSON system prompt response from cache",
			"system_prompt_length", len(systemPrompt),
			"user_prompt_length", len(userPrompt),
			"response_length", len(response),
			"duration_ms", time.Since(startTime).Milliseconds())
		return response, nil
	}
	
	c.logger.Debug("cache miss for JSON system prompt, calling underlying client",
		"system_prompt_length", len(systemPrompt),
		"user_prompt_length", len(userPrompt))
	
	response, err := c.AIClient.CompleteJSONWithSystem(ctx, systemPrompt, userPrompt)
	if err != nil {
		c.logger.Error("underlying client failed for JSON system prompt",
			"error", err)
		return "", err
	}
	
	if cacheErr := c.cache.Set(ctx, cacheKey, response); cacheErr != nil {
		c.logger.Warn("failed to cache JSON system prompt response",
			"error", cacheErr)
	}
	
	c.logger.Info("completed JSON system prompt with fresh response",
		"system_prompt_length", len(systemPrompt),
		"user_prompt_length", len(userPrompt),
		"response_length", len(response),
		"duration_ms", time.Since(startTime).Milliseconds())
	
	return response, nil
}