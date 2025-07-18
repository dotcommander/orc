# Configuration Guide

# Configuration Guide

This guide explains how to configure Orc for generating novels, code, and other content. Orc uses a simple YAML configuration file and environment variables.

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

**Default Locations**:
- Configuration is stored in `~/.config/orchestrator/`
- Generated content is saved to `~/.local/share/orchestrator/output/`
- Prompt templates are in `~/.local/share/orchestrator/prompts/`
- You can override any of these paths in the configuration

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

## Advanced Configuration Examples

### High-Performance Setup

For users with good API quotas who want faster generation:

```yaml
ai:
  timeout: 180  # Allow more time for complex requests

limits:
  max_concurrent_writers: 15  # More parallel processing
  max_retries: 5              # More resilient to failures
  phase_timeout: "45m"        # Allow longer for quality
  total_timeout: "6h"         # Support longer projects
```

### Conservative API Usage

For users who want to minimize API costs:

```yaml
limits:
  max_concurrent_writers: 2   # Minimal parallel requests
  max_retries: 1              # Fail fast to save costs
  phase_timeout: "15m"        # Shorter timeouts
  total_timeout: "2h"         # Limit total runtime
```

## Quick Configuration Examples

### Basic Setup for Novel Writing

```yaml
# ~/.config/orchestrator/config.yaml
ai:
  model: "claude-3-5-sonnet-20241022"  # Best for creative writing
  
limits:
  max_concurrent_writers: 8   # Good balance
  phase_timeout: "30m"        # Allow time for quality
```

### Setup for Code Generation

```yaml
# ~/.config/orchestrator/config.yaml
ai:
  model: "claude-3-5-sonnet-20241022"  # Great for code too
  timeout: 150                          # More time for complex code
  
features:
  enable_checkpointing: true  # Save progress for long projects
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

### Verifying Your Configuration

```bash
# Check if Orc can find your configuration
orc config get ai.model

# Test your API key is working
orc test-connection

# See all current settings
orc config list
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

## Next Steps

- Run `orc "Write a short story about robots"` to test your setup
- See [`performance.md`](performance.md) to optimize generation speed
- Check [`troubleshooting.md`](troubleshooting.md) if you encounter issues