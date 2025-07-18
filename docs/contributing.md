# Contributing Guide & Troubleshooting

**AI Context**: Complete development workflow, contribution guidelines, and error troubleshooting for The Orchestrator. Use this for implementing features, debugging, and resolving issues.

**Cross-references**: [`../CLAUDE.md`](../CLAUDE.md) for quick navigation, [`technical.md`](technical.md) for interfaces and architecture, [`performance.md`](performance.md) for optimization details.

## Getting Started

### Prerequisites
- **Go 1.21+** - Modern Go features and generics support
- **Make** - Build automation (optional but recommended)
- **Git** - Version control
- **Anthropic API Key** - For AI service access

### Development Environment Setup

```bash
# Clone the repository
git clone https://github.com/vampirenirmal/orchestrator.git
cd orc

# Install dependencies
make deps

# Copy and configure
cp config.yaml.example ~/.config/orchestrator/config.yaml
echo "OPENAI_API_KEY=your_key_here" > ~/.config/orchestrator/.env

# Run tests to verify setup
make test

# Build and install locally
make install
```

### Project Layout

```
orc/
├── cmd/orc/main.go               # Entry point - dependency injection only
├── internal/                     # Private application code
│   ├── core/                    # Core orchestration logic
│   │   ├── orchestrator.go      # Main orchestrator implementation
│   │   ├── execution_engine.go  # Extracted execution logic
│   │   ├── goal_orchestrator.go # Goal-aware orchestration
│   │   ├── strategies/          # Goal achievement strategies
│   │   ├── interfaces.go        # Core interfaces (Phase, Agent, Storage)
│   │   ├── checkpoint.go        # Resume functionality
│   │   └── errors.go            # Error types and handling
│   ├── domain/                  # Domain layer with plugins
│   │   └── plugin/              # Plugin implementations
│   ├── phases/                  # Phase implementations (legacy)
│   ├── agent/                   # AI service abstraction
│   ├── config/                  # Configuration management
│   ├── storage/                 # Storage abstraction
│   └── adapter/                 # Clean architecture adapters
├── prompts/                     # AI prompt templates
├── scripts/                     # Installation and utility scripts
├── docs/                        # Documentation (this directory)
└── config.yaml.example         # Configuration template
```

## Development Workflow

### 1. Feature Development Process

```bash
# 1. Create feature branch
git checkout -b feature/new-phase

# 2. Implement changes
# - Write tests first (TDD approach)
# - Implement feature
# - Update documentation

# 3. Verify implementation
make test          # Run all tests
make lint          # Run linters
make build         # Verify compilation

# 4. Test integration
make dev           # Run in development mode
orc -verbose "test prompt"

# 5. Update documentation
# - Update CLAUDE.md if interfaces change
# - Update README.md if user-facing features change
# - Add examples to docs/examples/

# 6. Commit and push
git add .
git commit -m "Add new phase implementation"
git push origin feature/new-phase
```

### 2. Testing Strategy

#### Unit Tests
```bash
# Run all tests with coverage
make test

# Run specific package tests
go test -v ./internal/agent
go test -v ./internal/core

# Run with race detection
go test -race ./...

# Run benchmarks
make bench
```

#### Integration Tests
```bash
# Full pipeline test
go test -v ./internal/core -run TestFullPipeline

# Specific phase integration
go test -v ./internal/phases/writing -run TestWriterConcurrency
```

#### Example Test Implementation
```go
// internal/phases/planning/planner_test.go
func TestPlannerExecute(t *testing.T) {
    // Setup
    mockAgent := agent.NewMockAgent()
    mockStorage := storage.NewMemoryStorage()
    planner := NewPlanner(mockAgent, mockStorage, "test-prompt.txt")
    
    // Configure mock responses
    mockAgent.SetResponse("planning prompt", `{
        "title": "Test Novel",
        "logline": "A test novel about testing",
        "chapters": [
            {"number": 1, "title": "Chapter 1", "summary": "Introduction"}
        ]
    }`)
    
    // Execute
    input := core.PhaseInput{
        Request: "Write a test novel",
        Data:    nil,
    }
    
    output, err := planner.Execute(context.Background(), input)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, output.Data)
    
    plan, ok := output.Data.(planning.NovelPlan)
    assert.True(t, ok)
    assert.Equal(t, "Test Novel", plan.Title)
    assert.Len(t, plan.Chapters, 1)
}
```

### 3. Code Quality Standards

#### Linting Configuration
```bash
# Install golangci-lint
make lint-deps

# Run all linters
make lint

# Fix auto-fixable issues
golangci-lint run --fix
```

#### Code Style Guidelines
- **Interface-first design** - Define interfaces before implementations
- **Dependency injection** - No global state, inject all dependencies
- **Error handling** - Use structured error types, wrap with context
- **Context propagation** - Pass context through all layers
- **Structured logging** - Use slog with consistent fields

## Common Development Tasks

### 1. Adding a New Phase

#### Step-by-Step Implementation
```bash
# 1. Create phase directory
mkdir -p internal/domain/plugin/phases/newphase

# 2. Create phase implementation
cat > internal/domain/plugin/phases/newphase/newphase.go << 'EOF'
package newphase

import (
    "context"
    "time"
    
    "github.com/vampirenirmal/orchestrator/internal/core"
)

type NewPhase struct {
    agent   core.Agent
    storage core.Storage
    config  Config
}

type Config struct {
    PromptPath string
    Timeout    time.Duration
}

func New(agent core.Agent, storage core.Storage, config Config) *NewPhase {
    return &NewPhase{
        agent:   agent,
        storage: storage,
        config:  config,
    }
}

func (p *NewPhase) Name() string {
    return "new-phase"
}

func (p *NewPhase) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
    // Implementation here
    return core.PhaseOutput{}, nil
}

func (p *NewPhase) ValidateInput(ctx context.Context, input core.PhaseInput) error {
    // Validation logic
    return nil
}

func (p *NewPhase) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
    // Output validation logic
    return nil
}

func (p *NewPhase) EstimatedDuration() time.Duration {
    return p.config.Timeout
}

func (p *NewPhase) CanRetry(err error) bool {
    // Retry logic
    return false
}
EOF

# 3. Create test file
cat > internal/domain/plugin/phases/newphase/newphase_test.go << 'EOF'
package newphase

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/vampirenirmal/orchestrator/internal/core"
)

func TestNewPhaseExecute(t *testing.T) {
    // Test implementation
}
EOF

# 4. Add to plugin implementation
# Edit internal/domain/plugin to include new phase in pipeline

# 5. Create prompt template
echo "New phase prompt template" > prompts/newphase.txt

# 6. Test implementation
go test -v ./internal/domain/plugin/phases/newphase
```

### 2. Modifying AI Interactions

#### Agent Enhancement
```go
// internal/agent/agent.go
func (a *Agent) ExecuteWithStreaming(ctx context.Context, prompt string, callback func(chunk string)) error {
    req := a.buildRequest(prompt)
    req.Stream = true
    
    resp, err := a.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        if err := ctx.Err(); err != nil {
            return err
        }
        
        chunk := scanner.Text()
        callback(chunk)
    }
    
    return scanner.Err()
}
```

### 3. Configuration Management

#### Adding New Configuration Options
```go
// internal/config/config.go
type Config struct {
    AI       AIConfig      `yaml:"ai" validate:"required"`
    Paths    PathsConfig   `yaml:"paths" validate:"required"`
    Limits   Limits        `yaml:"limits" validate:"required"`
    Features FeatureConfig `yaml:"features"` // New section
}

type FeatureConfig struct {
    EnableStreaming bool `yaml:"enable_streaming"`
    CacheTTL       time.Duration `yaml:"cache_ttl" validate:"min=1m"`
    Debug          bool `yaml:"debug"`
}

// Add defaults in validation
func (c *Config) validate() error {
    if c.Features.CacheTTL == 0 {
        c.Features.CacheTTL = 24 * time.Hour
    }
    
    // Existing validation...
    return nil
}
```

## Debugging and Troubleshooting

### 1. Debug Logging
```bash
# Enable debug logging
export REFINER_LOG_LEVEL=debug

# Run with verbose output
orc -verbose "test prompt"

# Check debug log
tail -f ~/.local/state/orchestrator/debug.log
```

### 2. Common Debug Patterns
```go
// Add debug logging to phases
func (p *MyPhase) Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error) {
    p.logger.Debug("phase execution starting",
        "phase", p.Name(),
        "input_size", len(fmt.Sprintf("%+v", input.Data)),
        "request", input.Request)
    
    start := time.Now()
    defer func() {
        p.logger.Debug("phase execution completed",
            "phase", p.Name(),
            "duration", time.Since(start))
    }()
    
    // Implementation...
}
```

## Error Catalog

### Build and Compilation Errors

#### Go Version Incompatibility

**Error Message:**
```
Go version X.X.X is too old. Minimum required: 1.21
```

**Root Cause:** The project requires Go 1.21 or later but an older version is installed.

**Solution:**
1. Update Go to version 1.21 or later
2. Download from https://golang.org/doc/install
3. Verify installation: `go version`

#### Missing Dependencies

**Error Message:**
```
package "github.com/go-playground/validator/v10" is not in GOROOT
module github.com/vampirenirmal/orchestrator: invalid version: unknown revision
```

**Solution:**
```bash
# Download and verify dependencies
go mod download
go mod verify

# Clean module cache if corrupted
go clean -modcache
go mod download
```

### Configuration Errors

#### Missing API Key

**Error Message:**
```
validating config: config validation failed: Field validation for 'APIKey' failed on the 'required' tag
```

**Solution:**
```bash
# Set via environment variable (recommended)
export OPENAI_API_KEY="your-anthropic-api-key"

# Or edit config file
vim ~/.config/orchestrator/config.yaml
# Set: api_key: "your-anthropic-api-key"
```

#### Invalid Configuration Values

**Allowed Ranges:**
- `ai.api_key`: minimum 20 characters
- `ai.model`: must be one of: `claude-3-5-sonnet-20241022`, `claude-3-opus-20240229`
- `ai.timeout`: 10-300 seconds
- `limits.max_concurrent_writers`: 1-100
- `limits.max_retries`: 0-10
- `limits.total_timeout`: 1 minute to 24 hours

### Runtime Errors

#### Missing Prompt Files

**Error Message:**
```
failed to preload prompts: reading prompt file: open /home/user/.local/share/orchestrator/prompts/orchestrator.txt: no such file or directory
```

**Solution:**
```bash
# Copy prompt files from source
mkdir -p ~/.local/share/orchestrator/prompts
cp prompts/*.txt ~/.local/share/orchestrator/prompts/

# Or reinstall completely
make install
```

#### Output Directory Permission Error

**Solution:**
```bash
# Change output directory to writable location
orc -output ~/novels "your request"

# Or fix permissions
sudo chmod 755 /path/to/output
sudo chown $USER:$USER /path/to/output
```

### API and Network Errors

#### Authentication Failure

**Error Message:**
```
API error (status 401): {"error": {"type": "authentication_error", "message": "invalid x-api-key"}}
```

**Solution:**
1. Verify API key at https://console.anthropic.com/
2. Update configuration:
   ```bash
   export OPENAI_API_KEY="your-new-api-key"
   ```

#### Rate Limiting

**Error Message:**
```
API error (status 429): {"error": {"type": "rate_limit_error", "message": "rate limit exceeded"}}
```

**Solution:**
```yaml
limits:
  rate_limit:
    requests_per_minute: 30  # Reduce from default 60
    burst_size: 5           # Reduce from default 10
```

### Performance Issues

#### High Memory Usage

**Solution:**
```yaml
limits:
  max_concurrent_writers: 5  # Reduce from default 10
```

#### Disk Space Issues

**Error Message:**
```
writing file: no space left on device
```

**Solution:**
```bash
# Change output directory to location with more space
orc -output /path/to/larger/disk "your request"

# Monitor disk usage
df -h
```

## Contributing Guidelines

### Pull Request Process
1. **Fork and branch** - Create feature branch from main
2. **Implement changes** - Follow code style guidelines
3. **Add tests** - Maintain or improve test coverage
4. **Update docs** - Keep documentation current
5. **Submit PR** - Include clear description and testing steps

### Code Review Checklist
- [ ] Follows interface-driven design
- [ ] Proper error handling with context
- [ ] Tests cover edge cases
- [ ] No breaking changes to public APIs
- [ ] Documentation updated
- [ ] Performance implications considered

### Git Commit Guidelines
```bash
# Commit message format
type(scope): description

# Examples
feat(agent): add streaming response support
fix(orchestrator): handle context cancellation properly
docs(api): update interface documentation
test(phases): add integration test for writing phase
```

### Build and Release Process

#### Build Automation
```makefile
# Makefile targets
.PHONY: build test lint deps install clean

# Build binary
build:
	go build -ldflags="-s -w" -o bin/orc ./cmd/orc

# Run tests with coverage
test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install locally (XDG compliant)
install: build
	mkdir -p ~/.local/bin
	cp bin/orc ~/.local/bin/
	chmod +x ~/.local/bin/orc
```

#### Release Checklist
- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Documentation updated
- [ ] Version bumped in code
- [ ] CHANGELOG.md updated
- [ ] Security review completed
- [ ] Performance benchmarks stable

## Troubleshooting Tips

### Enable Debug Logging

```bash
# Run with verbose logging
orc -verbose "your request"

# Check system logs
journalctl -u orc  # If running as service
tail -f ~/.local/share/orchestrator/logs/debug.log  # If debug logging enabled
```

### Check System Requirements

```bash
# Verify Go installation
go version

# Check available memory
free -h

# Check disk space
df -h ~/.local/share/orchestrator

# Verify network connectivity
ping api.anthropic.com
```

### Clean Installation

```bash
# Complete reinstall
make uninstall
rm -rf ~/.config/orchestrator ~/.local/share/orchestrator
make install

# Set up API key
export OPENAI_API_KEY="your-api-key"
echo 'export OPENAI_API_KEY="your-api-key"' >> ~/.bashrc
```

---

**Next Steps**: See [`technical.md`](technical.md) for detailed interfaces and architecture, [`performance.md`](performance.md) for optimization details, or [`examples/`](examples/) for usage patterns.

**Last Updated**: 2025-07-16