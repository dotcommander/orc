package plugin_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/domain"
	"github.com/vampirenirmal/orchestrator/internal/domain/plugin"
)

// Mock domain agent for testing
type mockDomainAgent struct {
	executeFunc     func(context.Context, string, any) (string, error)
	executeJSONFunc func(context.Context, string, any) (string, error)
}

func (m *mockDomainAgent) Execute(ctx context.Context, prompt string, input any) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, prompt, input)
	}
	return "mock response", nil
}

func (m *mockDomainAgent) ExecuteJSON(ctx context.Context, prompt string, input any) (string, error) {
	if m.executeJSONFunc != nil {
		return m.executeJSONFunc(ctx, prompt, input)
	}
	return `{"result": "mock json"}`, nil
}

// Mock domain storage for testing
type mockDomainStorage struct {
	data map[string][]byte
}

func newMockDomainStorage() *mockDomainStorage {
	return &mockDomainStorage{
		data: make(map[string][]byte),
	}
}

func (m *mockDomainStorage) Save(ctx context.Context, key string, data []byte) error {
	m.data[key] = data
	return nil
}

func (m *mockDomainStorage) Load(ctx context.Context, key string) ([]byte, error) {
	data, ok := m.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return data, nil
}

func (m *mockDomainStorage) Exists(ctx context.Context, key string) bool {
	_, ok := m.data[key]
	return ok
}

// Test domain plugin implementation
type testDomainPlugin struct {
	agent   domain.Agent
	storage domain.Storage
}

func NewTestDomainPlugin(agent domain.Agent, storage domain.Storage) *testDomainPlugin {
	return &testDomainPlugin{
		agent:   agent,
		storage: storage,
	}
}

func (p *testDomainPlugin) Name() string {
	return "test"
}

func (p *testDomainPlugin) Description() string {
	return "Test plugin for validation"
}

func (p *testDomainPlugin) GetPhases() []domain.Phase {
	return []domain.Phase{
		&mockDomainPhase{
			name: "TestPhase",
			executeFunc: func(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
				// Use the actual agent to test failures
				_, err := p.agent.Execute(ctx, "test prompt", input.Request)
				if err != nil {
					return domain.PhaseOutput{}, err
				}
				return domain.PhaseOutput{
					Data: map[string]interface{}{
						"result": "test phase completed",
						"input":  input.Request,
					},
				}, nil
			},
			validateInputFunc: func(ctx context.Context, input domain.PhaseInput) error {
				return nil
			},
			validateOutputFunc: func(ctx context.Context, output domain.PhaseOutput) error {
				return nil
			},
		},
	}
}

func (p *testDomainPlugin) GetDefaultConfig() plugin.DomainPluginConfig {
	return plugin.DomainPluginConfig{}
}

func (p *testDomainPlugin) ValidateRequest(request string) error {
	if strings.TrimSpace(request) == "" {
		return errors.New("request cannot be empty")
	}
	return nil
}

func (p *testDomainPlugin) GetOutputSpec() plugin.DomainOutputSpec {
	return plugin.DomainOutputSpec{
		PrimaryOutput: "test_output.txt",
	}
}

func (p *testDomainPlugin) GetDomainValidator() domain.DomainValidator {
	return &mockDomainValidator{}
}

// Mock domain validator
type mockDomainValidator struct{}

func (v *mockDomainValidator) ValidateRequest(request string) error {
	return nil
}

func (v *mockDomainValidator) ValidateOutput(output interface{}) error {
	return nil
}

func (v *mockDomainValidator) ValidatePhaseTransition(from, to string, data interface{}) error {
	return nil
}

// Mock domain phase for testing
type mockDomainPhase struct {
	name               string
	executeFunc        func(context.Context, domain.PhaseInput) (domain.PhaseOutput, error)
	validateInputFunc  func(context.Context, domain.PhaseInput) error
	validateOutputFunc func(context.Context, domain.PhaseOutput) error
	estimatedDuration  time.Duration
}

func (m *mockDomainPhase) Name() string {
	return m.name
}

func (m *mockDomainPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return domain.PhaseOutput{Data: "mock output"}, nil
}

func (m *mockDomainPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if m.validateInputFunc != nil {
		return m.validateInputFunc(ctx, input)
	}
	return nil
}

func (m *mockDomainPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	if m.validateOutputFunc != nil {
		return m.validateOutputFunc(ctx, output)
	}
	return nil
}

func (m *mockDomainPhase) EstimatedDuration() time.Duration {
	if m.estimatedDuration > 0 {
		return m.estimatedDuration
	}
	return time.Minute
}

func (m *mockDomainPhase) CanRetry(err error) bool {
	return true
}

func (m *mockDomainStorage) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockDomainStorage) List(ctx context.Context, pattern string) ([]string, error) {
	var results []string
	for key := range m.data {
		results = append(results, key)
	}
	return results, nil
}

func TestFictionPlugin(t *testing.T) {
	agent := &mockDomainAgent{}
	storage := newMockDomainStorage()

	fictionPlugin := plugin.NewFictionPlugin(agent, storage)

	// Test Name
	if fictionPlugin.Name() != "fiction" {
		t.Errorf("expected name 'fiction', got %s", fictionPlugin.Name())
	}

	// Test Description
	description := fictionPlugin.Description()
	if !strings.Contains(description, "novel") || !strings.Contains(description, "generation") {
		t.Errorf("expected description to mention novel generation, got: %s", description)
	}

	// Test GetDefaultConfig
	config := fictionPlugin.GetDefaultConfig()
	if len(config.Prompts) == 0 {
		t.Error("expected default config to have prompts")
	}

	// Test GetOutputSpec
	outputSpec := fictionPlugin.GetOutputSpec()
	if outputSpec.PrimaryOutput != "complete_novel.md" {
		t.Errorf("expected primary output 'complete_novel.md', got %s", outputSpec.PrimaryOutput)
	}

	if len(outputSpec.SecondaryOutputs) == 0 {
		t.Error("expected secondary outputs to be defined")
	}

	// Test ValidateRequest - valid fiction request
	err := fictionPlugin.ValidateRequest("Write a sci-fi novel about space exploration")
	if err != nil {
		t.Errorf("expected valid fiction request to pass, got error: %v", err)
	}

	// Test ValidateRequest - invalid request (too short)
	err = fictionPlugin.ValidateRequest("short")
	if err == nil {
		t.Error("expected short request to fail validation")
	}

	// Test ValidateRequest - non-fiction request
	err = fictionPlugin.ValidateRequest("Build an API for user authentication")
	if err == nil {
		t.Error("expected code request to fail fiction validation")
	}
}

func TestCodePlugin(t *testing.T) {
	agent := &mockDomainAgent{}
	storage := newMockDomainStorage()

	codePlugin := plugin.NewCodePlugin(agent, storage)

	// Test Name
	if codePlugin.Name() != "code" {
		t.Errorf("expected name 'code', got %s", codePlugin.Name())
	}

	// Test Description
	description := codePlugin.Description()
	if !strings.Contains(description, "code") || !strings.Contains(description, "generation") {
		t.Errorf("expected description to mention code generation, got: %s", description)
	}

	// Test GetDefaultConfig
	config := codePlugin.GetDefaultConfig()
	if len(config.Prompts) == 0 {
		t.Error("expected default config to have prompts")
	}

	// Test GetOutputSpec
	outputSpec := codePlugin.GetOutputSpec()
	if outputSpec.PrimaryOutput != "code_output.md" {
		t.Errorf("expected primary output 'code_output.md', got %s", outputSpec.PrimaryOutput)
	}

	// Test ValidateRequest - valid code request
	err := codePlugin.ValidateRequest("Create a REST API for user management in Go")
	if err != nil {
		t.Errorf("expected valid code request to pass, got error: %v", err)
	}

	// Test ValidateRequest - invalid request (too short)
	err = codePlugin.ValidateRequest("short")
	if err == nil {
		t.Error("expected short request to fail validation")
	}

	// Test ValidateRequest - fiction request  
	err = codePlugin.ValidateRequest("Tell a romantic story about magical creatures")
	if err == nil {
		t.Error("expected fiction request to fail code validation")
	}
}

func TestFictionValidator(t *testing.T) {
	validator := &plugin.FictionValidator{}

	// Test ValidateRequest - valid cases
	validRequests := []string{
		"Write a novel about space exploration",
		"Create a story with fantasy elements",
		"Develop characters for a thriller plot",
		"Write a romance book",
		"Create a sci-fi narrative",
	}

	for _, request := range validRequests {
		err := validator.ValidateRequest(request)
		if err != nil {
			t.Errorf("expected valid request '%s' to pass, got error: %v", request, err)
		}
	}

	// Test ValidateRequest - invalid cases
	invalidRequests := []string{
		"", // empty
		"Build a REST API for authentication", // code-related
		"Build a database schema for users",     // technical
		"Debug the API server connection",     // programming
	}

	for _, request := range invalidRequests {
		err := validator.ValidateRequest(request)
		if err == nil {
			t.Errorf("expected invalid request '%s' to fail validation", request)
		}
	}

	// Test ValidatePhaseTransition
	tests := []struct {
		name     string
		from     string
		to       string
		data     interface{}
		wantErr  bool
	}{
		{
			name: "valid planning to architecture",
			from: "Planning",
			to:   "Architecture",
			data: map[string]interface{}{
				"title": "Test Novel",
				"plot":  "A story about...",
			},
			wantErr: false,
		},
		{
			name: "invalid planning to architecture - missing title",
			from: "Planning",
			to:   "Architecture",
			data: map[string]interface{}{
				"plot": "A story about...",
			},
			wantErr: true,
		},
		{
			name: "valid architecture to writing",
			from: "Architecture",
			to:   "Writing",
			data: map[string]interface{}{
				"characters": []string{"Alice", "Bob"},
				"settings":   []string{"New York", "Mars"},
			},
			wantErr: false,
		},
		{
			name: "invalid architecture to writing - missing characters",
			from: "Architecture",
			to:   "Writing",
			data: map[string]interface{}{
				"settings": []string{"New York", "Mars"},
			},
			wantErr: true,
		},
		{
			name: "nil data",
			from: "Planning",
			to:   "Architecture",
			data: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePhaseTransition(tt.from, tt.to, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePhaseTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCodeValidator(t *testing.T) {
	validator := &plugin.CodeValidator{}

	// Test ValidateRequest - valid cases
	validRequests := []string{
		"Create a REST API for user management",
		"Build a Python script for data processing",
		"Implement a Go microservice",
		"Write JavaScript functions for validation",
		"Develop a database schema",
		"Create unit tests for the service",
		"Refactor the authentication module",
		"Debug the payment processing logic",
	}

	for _, request := range validRequests {
		err := validator.ValidateRequest(request)
		if err != nil {
			t.Errorf("expected valid request '%s' to pass, got error: %v", request, err)
		}
	}

	// Test ValidateRequest - invalid cases
	invalidRequests := []string{
		"", // empty
		"Write a romantic novel about developers", // fiction
		"Create a story with fantasy elements",    // fiction
		"Develop characters for a thriller plot", // fiction
	}

	for _, request := range invalidRequests {
		err := validator.ValidateRequest(request)
		if err == nil {
			t.Errorf("expected invalid request '%s' to fail validation", request)
		}
	}

	// Test ValidatePhaseTransition
	tests := []struct {
		name     string
		from     string
		to       string
		data     interface{}
		wantErr  bool
	}{
		{
			name: "valid analysis to planning",
			from: "Analysis",
			to:   "Planning",
			data: map[string]interface{}{
				"complexity": "medium",
				"language":   "Go",
			},
			wantErr: false,
		},
		{
			name: "invalid analysis to planning - missing complexity",
			from: "Analysis",
			to:   "Planning",
			data: map[string]interface{}{
				"language": "Go",
			},
			wantErr: true,
		},
		{
			name: "valid planning to implementation",
			from: "Planning",
			to:   "Implementation",
			data: map[string]interface{}{
				"steps": []string{"step1", "step2"},
				"files": []string{"main.go", "handler.go"},
			},
			wantErr: false,
		},
		{
			name: "invalid planning to implementation - missing files",
			from: "Planning",
			to:   "Implementation",
			data: map[string]interface{}{
				"steps": []string{"step1", "step2"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePhaseTransition(tt.from, tt.to, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePhaseTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPluginAdapter(t *testing.T) {
	agent := &mockDomainAgent{}
	storage := newMockDomainStorage()

	// Test with fiction plugin
	fictionDomain := plugin.NewFictionPlugin(agent, storage)
	fictionAdapter := plugin.NewPluginAdapter(fictionDomain)

	// Test adapter delegates to domain plugin
	if fictionAdapter.Name() != fictionDomain.Name() {
		t.Errorf("adapter name mismatch: expected %s, got %s", fictionDomain.Name(), fictionAdapter.Name())
	}

	if fictionAdapter.Description() != fictionDomain.Description() {
		t.Errorf("adapter description mismatch")
	}

	// Test config conversion
	adapterConfig := fictionAdapter.GetDefaultConfig()
	domainConfig := fictionDomain.GetDefaultConfig()

	if len(adapterConfig.Prompts) != len(domainConfig.Prompts) {
		t.Errorf("config prompts count mismatch: expected %d, got %d", 
			len(domainConfig.Prompts), len(adapterConfig.Prompts))
	}

	// Test output spec conversion
	adapterSpec := fictionAdapter.GetOutputSpec()
	domainSpec := fictionDomain.GetOutputSpec()

	if adapterSpec.PrimaryOutput != domainSpec.PrimaryOutput {
		t.Errorf("output spec primary mismatch: expected %s, got %s", 
			domainSpec.PrimaryOutput, adapterSpec.PrimaryOutput)
	}

	// Test validation
	err := fictionAdapter.ValidateRequest("Write a science fiction novel")
	if err != nil {
		t.Errorf("adapter validation failed: %v", err)
	}

	// Test version (should have default)
	if fictionAdapter.Version() == "" {
		t.Error("adapter should have a version")
	}
}

// TestDomainPluginRunnerExecution tests the complete execution flow
func TestDomainPluginRunnerExecution(t *testing.T) {
	// Create mock dependencies
	agent := &mockDomainAgent{
		executeFunc: func(ctx context.Context, prompt string, input any) (string, error) {
			return "Mock AI response", nil
		},
	}
	storage := newMockDomainStorage()

	// Create test plugin
	testPlugin := NewTestDomainPlugin(agent, storage)
	
	// Create registry and register plugin
	registry := plugin.NewDomainRegistry()
	err := registry.Register(testPlugin)
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	// Create plugin runner
	runner := plugin.NewDomainPluginRunner(registry, storage)

	// Test successful execution
	t.Run("successful execution", func(t *testing.T) {
		err := runner.Execute(context.Background(), "test", "Create a test project")
		if err != nil {
			t.Errorf("execution failed: %v", err)
		}

		// Verify storage was used
		if len(storage.data) == 0 {
			t.Error("no data was saved to storage during execution")
		}
	})

	// Test plugin not found
	t.Run("plugin not found", func(t *testing.T) {
		err := runner.Execute(context.Background(), "nonexistent", "test request")
		if err == nil {
			t.Error("expected error for nonexistent plugin")
		}

		var notFoundErr *plugin.DomainPluginNotFoundError
		if !errors.As(err, &notFoundErr) {
			t.Errorf("expected DomainPluginNotFoundError, got %T", err)
		}
	})

	// Test invalid request
	t.Run("invalid request", func(t *testing.T) {
		err := runner.Execute(context.Background(), "test", "") // empty request
		if err == nil {
			t.Error("expected error for invalid request")
		}

		var invalidErr *plugin.DomainInvalidRequestError
		if !errors.As(err, &invalidErr) {
			t.Errorf("expected DomainInvalidRequestError, got %T", err)
		}
	})

	// Test phase execution failure
	t.Run("phase execution failure", func(t *testing.T) {
		// Create agent that returns errors
		failingAgent := &mockDomainAgent{
			executeFunc: func(ctx context.Context, prompt string, input any) (string, error) {
				return "", errors.New("AI service unavailable")
			},
		}
		
		failingPlugin := NewTestDomainPlugin(failingAgent, storage)
		failingRegistry := plugin.NewDomainRegistry()
		failingRegistry.Register(failingPlugin)
		failingRunner := plugin.NewDomainPluginRunner(failingRegistry, storage)

		err := failingRunner.Execute(context.Background(), "test", "Create a test project")
		if err == nil {
			t.Error("expected error for failing phase execution")
		}

		var execErr *plugin.DomainPhaseExecutionError
		if !errors.As(err, &execErr) {
			t.Errorf("expected DomainPhaseExecutionError, got %T", err)
		}
	})
}