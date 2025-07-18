# Troubleshooting Guide

This guide helps you resolve common issues when using Orc.

## Quick Troubleshooting

Before diving into specific issues, check these three things:

1. **API Key**: Is your `OPENAI_API_KEY` environment variable set?
   ```bash
   echo $OPENAI_API_KEY  # Should show your key (partially hidden)
   ```

2. **Configuration**: Does Orc find your config?
   ```bash
   orc config list  # Shows current configuration
   ```

3. **Connection**: Can Orc reach the AI service?
   ```bash
   orc test-connection  # Tests API connectivity
   ```

## Configuration Issues

### "API key not found"

**What you'll see:**
```
Error: AI API key not provided
```

**Solution:**
1. Set the environment variable:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```
2. Or add to `~/.config/orchestrator/.env`:
   ```
   OPENAI_API_KEY=your-api-key-here
   ```
3. Or add to `~/.config/orchestrator/config.yaml`:
   ```yaml
   ai:
     api_key: "your-api-key-here"
   ```

### "Config file not found"

**What you'll see:**
```
Error: Failed to load configuration
```

**Solution:**
1. Create the config directory:
   ```bash
   mkdir -p ~/.config/orchestrator
   ```
2. Copy the example config:
   ```bash
   cp config.yaml.example ~/.config/orchestrator/config.yaml
   ```
3. Or specify a custom config:
   ```bash
   orc -config /path/to/your/config.yaml "Write a story"
   ```

### "Invalid model specified"

**What you'll see:**
```
Error: Model 'gpt-5' is not supported
```

**Solution:**
Use a supported model in your config:
```yaml
ai:
  model: "claude-3-5-sonnet-20241022"  # Recommended
  # or: "claude-3-opus-20240229"
  # or: "gpt-4"
```

## Generation Issues

### "Generation timed out"

**What you'll see:**
```
Error: Phase timeout exceeded (30m0s)
```

**Solution:**
1. Increase timeouts in config:
   ```yaml
   limits:
     phase_timeout: "45m"
     total_timeout: "6h"
   ```
2. Or use shorter prompts for faster generation
3. Try with `--fluid` mode for better handling

### "AI generated wrong language"

**What you'll see:**
```
Error: Expected PHP code but received JavaScript
```

**Solution:**
Be explicit in your prompt:
```bash
# Be very specific
orc create code "ONLY USE PHP. No JavaScript. Create a PHP REST API"

# Use language constraints
orc create code "Create a Python web scraper using Python only"
```

### "Content is too short"

**What you'll see:**
```
Verification failed: Chapter 3 only has 234 words (minimum: 1000)
```

**Solution:**
1. Use fluid mode for better results:
   ```bash
   orc create fiction "Write a novel" --fluid
   ```
2. Increase generation time:
   ```yaml
   ai:
     timeout: 180  # 3 minutes per request
   ```
3. Be specific about length in your prompt

### "Rate limit exceeded"

**What you'll see:**
```
Error: API rate limit exceeded. Please retry after 60s
```

**Solution:**
1. Wait for the cooldown period
2. Reduce concurrent operations:
   ```yaml
   limits:
     max_concurrent_writers: 3  # Lower number
   ```
3. Check your API plan limits

## Output Issues

### "Permission denied saving output"

**What you'll see:**
```
Error: Failed to create output directory: permission denied
```

**Solution:**
1. Check directory permissions:
   ```bash
   ls -la ~/.local/share/orchestrator/
   ```
2. Fix permissions:
   ```bash
   chmod 755 ~/.local/share/orchestrator
   mkdir -p ~/.local/share/orchestrator/output
   ```
3. Or use a different output directory:
   ```bash
   orc -output ~/Documents/my-novels "Write a story"
   ```

### "Disk space full"

**What you'll see:**
```
Error: No space left on device
```

**Solution:**
1. Check available space:
   ```bash
   df -h ~/.local/share/orchestrator
   ```
2. Clean old sessions:
   ```bash
   rm -rf ~/.local/share/orchestrator/output/old-sessions/
   ```
3. Use a different disk:
   ```yaml
   paths:
     output_dir: "/external-drive/orc-output"
   ```

## Session and Resume Issues

### "Session not found"

**What you'll see:**
```
Error: Session 'abc123' not found
```

**Solution:**
1. List available sessions:
   ```bash
   ls ~/.local/share/orchestrator/output/sessions/
   ```
2. Check if you have the right session ID
3. Sessions older than 30 days may be auto-cleaned

### "Cannot resume - checkpoint corrupted"

**What you'll see:**
```
Error: Failed to load checkpoint data
```

**Solution:**
1. Try resuming from an earlier checkpoint
2. Start fresh with the same prompt
3. Check for partial output you can use

## Common Patterns

### Getting Help

**Quick diagnostics:**
```bash
# Check version
orc --version

# Test configuration
orc config validate

# See detailed logs
orc create fiction "Test" --verbose

# Check debug log
tail -f ~/.local/state/orchestrator/debug.log
```

### Best Practices to Avoid Issues

1. **Always use explicit language in prompts**
   - Bad: "Create a web app"
   - Good: "Create a React web app with TypeScript"

2. **Start small, then scale up**
   - Test with short content first
   - Increase complexity gradually

3. **Monitor your API usage**
   - Check rate limits
   - Watch your monthly quota

4. **Use appropriate timeouts**
   - Fiction: 30-45 minutes per phase
   - Code: 15-30 minutes per phase

5. **Save your work**
   - Use version control for outputs
   - Keep session IDs for resuming

## Still Having Issues?

If you're still experiencing problems:

1. **Check the error log**:
   ```bash
   tail -n 50 ~/.local/state/orchestrator/error.log
   ```

2. **Run with verbose logging**:
   ```bash
   orc --verbose create fiction "Test prompt"
   ```

3. **Verify your environment**:
   ```bash
   orc doctor  # Runs diagnostic checks
   ```

4. **Report issues**:
   - Include the full error message
   - Share your config (without API keys)
   - Provide the command you ran

## Error Prevention Checklist

Before running Orc:
- [ ] API key is set and valid
- [ ] Config file exists and is valid YAML
- [ ] Output directory is writable
- [ ] Sufficient disk space (at least 1GB)
- [ ] Network connection is stable
- [ ] Using a supported AI model

This should help you resolve most common issues with Orc!