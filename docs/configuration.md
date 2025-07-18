# Configuration Guide

**AI Context**: Complete configuration reference for The Orchestrator. Use this for understanding all configuration options, XDG compliance, and environment setup.

**Cross-references**: [`../CLAUDE.md`](../CLAUDE.md) for quick reference, [`../README.md`](../README.md) for basic setup, [`development.md`](development.md) for development configuration, [`paths.md`](paths.md) for file locations, [`errors.md`](errors.md) for configuration troubleshooting.

## Overview

The Orchestrator follows the **XDG Base Directory Specification** for configuration management, providing a clean and predictable configuration experience across different environments.

### Configuration Priority

Configuration values are resolved in the following order (highest to lowest priority):

1. **Command-line flags** (`-config`, `-output`, `-verbose`)
2. **Environment variables** (`OPENAI_API_KEY`, `ORC_CONFIG`)
3. **XDG config file** (`~/.config/orchestrator/config.yaml`)
4. **Built-in defaults**

## File Locations (XDG Compliant)

### Configuration Files
```bash
# Primary config file
~/.config/orchestrator/config.yaml

# Environment variables file
~/.config/orchestrator/.env

# Alternative with XDG_CONFIG_HOME
$XDG_CONFIG_HOME/orchestrator/config.yaml
$XDG_CONFIG_HOME/orchestrator/.env
```

### Data Files
```bash
# Prompt templates
~/.local/share/orchestrator/prompts/

# Output novels
~/.local/share/orchestrator/output/

# Alternative with XDG_DATA_HOME
$XDG_DATA_HOME/orchestrator/prompts/
$XDG_DATA_HOME/orchestrator/output/
```

### Runtime Files
```bash
# Debug and error logs
~/.local/state/orchestrator/debug.log
~/.local/state/orchestrator/error.log

# Alternative with XDG_STATE_HOME
$XDG_STATE_HOME/orchestrator/debug.log
$XDG_STATE_HOME/orchestrator/error.log
```

### Binary Installation
```bash
# Executable location
~/.local/bin/orchestrator

# Go binary symlink (preferred)
~/go/bin/orchestrator -> ~/.local/bin/orchestrator
```

## Configuration File Structure

### Complete Configuration Example

```yaml
# ~/.config/orchestrator/config.yaml

# AI Service Configuration
ai:
  api_key: ""  # Leave empty - use environment variable
  model: "claude-3-5-sonnet-20241022"
  base_url: "https://api.anthropic.com"
  timeout: 120  # seconds

# File Path Configuration
paths:
  output_dir: ""  # Defaults to ~/.local/share/orchestrator/output
  prompts:
    orchestrator: ""  # Defaults to ~/.local/share/orchestrator/prompts/orchestrator.txt
    architect: ""     # Defaults to ~/.local/share/orchestrator/prompts/architect.txt
    writer: ""        # Defaults to ~/.local/share/orchestrator/prompts/writer.txt
    critic: ""        # Defaults to ~/.local/share/orchestrator/prompts/critic.txt

# Resource Limits
limits:
  max_concurrent_writers: 10
  max_retries: 3
  phase_timeout: "30m"
  total_timeout: "4h"

# Logging Configuration
log:
  level: "info"  # debug, info, warn, error
  format: "text"  # text, json
  file: ""  # Empty = stdout, or path to log file

# Feature Flags
features:
  enable_checkpointing: true
  enable_caching: true
  cache_ttl: "24h"
  debug_mode: false
```

### Minimal Configuration

For basic usage, you only need to set the API key:

```yaml
# ~/.config/orchestrator/config.yaml
ai:
  api_key: "your-api-key-here"
```

Or better yet, use an environment variable:

```bash
# ~/.config/orchestrator/.env
OPENAI_API_KEY=your-api-key-here
```

## Environment Variables

### Required Variables
```bash
# AI service authentication
OPENAI_API_KEY=your_anthropic_api_key_here
```

### Optional Override Variables
```bash
# XDG directory overrides
XDG_CONFIG_HOME=/custom/config/path
XDG_DATA_HOME=/custom/data/path
XDG_STATE_HOME=/custom/state/path

# The Orchestrator-specific overrides
ORC_CONFIG=/path/to/custom/config.yaml
ORC_OUTPUT_DIR=/path/to/custom/output
ORC_LOG_LEVEL=debug
```

## Configuration Sections

### AI Configuration (`ai`)

```yaml
ai:
  api_key: ""                              # API key for AI service
  model: "claude-3-5-sonnet-20241022"      # AI model to use
  base_url: "https://api.anthropic.com"    # API endpoint
  timeout: 120                             # Request timeout in seconds
```

**Supported Models**:
- `claude-3-5-sonnet-20241022` (recommended, balanced performance)
- `claude-3-opus-20240229` (highest quality, slower)
- `gpt-4` (OpenAI alternative, requires different base_url)

**Model Selection Guidelines**:
- **claude-3-5-sonnet**: Best balance of speed and quality for novel generation
- **claude-3-opus**: Use for highest quality output when time is not critical
- **gpt-4**: Alternative provider option

### Paths Configuration (`paths`)

```yaml
paths:
  output_dir: "/custom/output/path"        # Where novels are saved
  prompts:
    orchestrator: "/path/to/orchestrator.txt"
    architect: "/path/to/architect.txt"
    writer: "/path/to/writer.txt"
    critic: "/path/to/critic.txt"
```

**Path Resolution**:
- **Relative paths**: Resolved relative to config file location
- **Absolute paths**: Used as-is
- **Empty values**: Use XDG-compliant defaults
- **`~` expansion**: Home directory is expanded automatically

### Limits Configuration (`limits`)

```yaml
limits:
  max_concurrent_writers: 10      # Writer pool size (1-20)
  max_retries: 3                  # Per-phase retry attempts (1-10)
  phase_timeout: "30m"            # Individual phase timeout
  total_timeout: "4h"             # Total pipeline timeout
```

**Performance Tuning**:
- **max_concurrent_writers**: Higher values = faster writing but more API load
  - 5-10: Good for most use cases
  - 15-20: High-performance setups with good API quotas
  - 1-3: Conservative API usage
- **max_retries**: Balance between resilience and speed
  - 3: Default, good for most scenarios
  - 5-10: Unreliable network conditions
  - 1: Fast failure for debugging

### Logging Configuration (`log`)

```yaml
log:
  level: "info"                   # debug, info, warn, error
  format: "text"                  # text, json
  file: "/path/to/logfile.log"    # Empty = stdout
```

**Log Levels**:
- **debug**: Verbose output, includes AI prompts and responses
- **info**: Normal operation information
- **warn**: Warning conditions, recoverable errors
- **error**: Error conditions only

**Log Formats**:
- **text**: Human-readable format for development
- **json**: Structured format for log aggregation

## Command-Line Configuration

### Flag-Based Configuration

```bash
# Override configuration file
orc -config /path/to/config.yaml "write a novel"

# Override output directory
orc -output /tmp/my-novel "write a mystery"

# Enable verbose logging
orc -verbose "write a sci-fi novel"

# Resume from checkpoint
orc -resume session-12345 "continue previous novel"

# Combine multiple flags
orc -config custom.yaml -output ./novels -verbose "write a fantasy novel"
```

### Environment Variable Usage

```bash
# Set API key
export OPENAI_API_KEY="your-key-here"

# Override default config path
export ORC_CONFIG="/path/to/custom/config.yaml"

# Set XDG directories
export XDG_CONFIG_HOME="/custom/config"
export XDG_DATA_HOME="/custom/data"

# Run with environment
orc "write a novel about environment variables"
```

## Configuration Validation

### Automatic Validation

The Orchestrator validates configuration on startup using structured validation:

```yaml
# Example validation errors
ai.api_key: required field missing
ai.timeout: must be between 10 and 300 seconds
limits.max_concurrent_writers: must be between 1 and 20
paths.output_dir: directory must be writable
```

### Manual Validation

```bash
# Test configuration without running
orc -config /path/to/config.yaml -validate

# Check specific configuration values
orc -config /path/to/config.yaml -show-config
```

## Development Configuration

### Development-Specific Settings

```yaml
# Development config example
ai:
  api_key: "test-key"
  timeout: 30  # Shorter timeouts for testing

paths:
  output_dir: "./test-output"
  prompts:
    orchestrator: "./prompts/orchestrator.txt"
    architect: "./prompts/architect.txt"
    writer: "./prompts/writer.txt"
    critic: "./prompts/critic.txt"

limits:
  max_concurrent_writers: 2  # Lower for debugging
  max_retries: 1            # Fail fast in development
  phase_timeout: "5m"       # Shorter timeouts
  total_timeout: "30m"

log:
  level: "debug"            # Verbose logging
  format: "text"            # Human-readable
  file: "./debug.log"       # Local log file

features:
  debug_mode: true          # Enable debug features
  enable_caching: false     # Disable caching for testing
```

### Testing Configuration

```yaml
# Testing config for CI/CD
ai:
  api_key: "${OPENAI_API_KEY}"  # From environment
  model: "claude-3-5-sonnet-20241022"
  timeout: 60

paths:
  output_dir: "/tmp/orchestrator-test"

limits:
  max_concurrent_writers: 3
  max_retries: 2
  phase_timeout: "10m"
  total_timeout: "1h"

log:
  level: "warn"             # Minimal logging in tests
  format: "json"            # Structured for parsing
```

## Configuration Migration

### Upgrading from Previous Versions

```bash
# Backup existing configuration
cp ~/.config/orchestrator/config.yaml ~/.config/orchestrator/config.yaml.backup

# Update configuration format (if needed)
orc -migrate-config

# Validate new configuration
orc -validate
```

### Configuration Templates

```bash
# Generate default configuration
orc -generate-config > ~/.config/orchestrator/config.yaml

# Generate development configuration
orc -generate-config -dev > config-dev.yaml

# Generate production configuration
orc -generate-config -prod > config-prod.yaml
```

## Troubleshooting Configuration

### Common Configuration Issues

1. **API Key Not Found**
   ```bash
   Error: AI API key not provided
   Solution: Set OPENAI_API_KEY environment variable or add to config
   ```

2. **Permission Denied**
   ```bash
   Error: Failed to create output directory
   Solution: Check directory permissions and XDG path setup
   ```

3. **Invalid Configuration**
   ```bash
   Error: Config validation failed
   Solution: Run 'orc -validate' for detailed error messages
   ```

4. **File Not Found**
   ```bash
   Error: Prompt file not found
   Solution: Check prompt paths in configuration and ensure files exist
   ```

### Debug Configuration Loading

```bash
# Show effective configuration
orc -show-config

# Show configuration sources
orc -debug-config

# Validate configuration
orc -validate

# Test configuration with dry run
orc -dry-run "test prompt"
```

## Security Considerations

### API Key Management

```bash
# ✅ Good: Environment variable
export OPENAI_API_KEY="your-key-here"

# ✅ Good: Separate .env file with restricted permissions
echo "OPENAI_API_KEY=your-key-here" > ~/.config/orchestrator/.env
chmod 600 ~/.config/orchestrator/.env

# ❌ Bad: API key in config file
# api_key: "your-key-here"  # Don't do this
```

### File Permissions

```bash
# Set secure permissions on configuration
chmod 600 ~/.config/orchestrator/config.yaml
chmod 600 ~/.config/orchestrator/.env

# Ensure directories have correct permissions
chmod 700 ~/.config/orchestrator
chmod 755 ~/.local/share/orchestrator
```

### Path Security

- **Avoid absolute paths** in shareable configurations
- **Use XDG variables** for portable configurations
- **Validate paths** to prevent directory traversal
- **Restrict output directories** to safe locations

---

**Next Steps**: See [`troubleshooting.md`](troubleshooting.md) for configuration-related issues, [`development.md`](development.md) for development setup, or [`../README.md`](../README.md) for basic usage.

**Last Updated**: 2025-07-16