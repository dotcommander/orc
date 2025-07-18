# Orchestrator Troubleshooting Guide

**Last Updated**: 2025-07-18  
**Purpose**: Help users resolve common issues when using the Orchestrator  
**Audience**: End users running `orc` commands

## Quick Troubleshooting

### Before You Start
Most issues can be resolved by checking these three things:
1. **API Key**: Is your OpenAI API key set? Run: `echo $OPENAI_API_KEY`
2. **Configuration**: Does your config file exist? Check: `~/.config/orchestrator/config.yaml`
3. **Permissions**: Can you write to output directories? Test: `ls -la ~/.local/share/orchestrator/`

## Common Issues by Task

### 🔧 Configuration Errors

#### "API key not configured"
**What you'll see**: Error message about missing API key when trying to generate content

**Solutions**:
1. Set your OpenAI API key:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```
2. Add it to your shell profile (`.bashrc`, `.zshrc`, etc.) to make it permanent
3. Verify it's set: `echo $OPENAI_API_KEY`

#### "Configuration file not found"
**What you'll see**: Error about missing config file at startup

**Solutions**:
1. Copy the example configuration:
   ```bash
   cp config.yaml.example ~/.config/orchestrator/config.yaml
   ```
2. Edit the config file to match your preferences:
   ```bash
   nano ~/.config/orchestrator/config.yaml
   ```

#### "Wrong AI model being used"
**What you'll see**: Logs show "gpt-4" when you configured "gpt-4.1" or another model

**Solutions**:
1. Check your config file:
   ```bash
   grep "model:" ~/.config/orchestrator/config.yaml
   ```
2. Make sure the model name is exactly what you want (including version numbers)
3. The orchestrator will use whatever model you specify - it won't "correct" it

### 🚀 Generation Errors

#### "Generation completed too quickly with poor results"
**What you'll see**: Content generation finishes in 30-60 seconds but quality is low

**Solutions**:
1. Use the `--fluid` flag for better quality:
   ```bash
   ./orc create code "your request" --fluid --verbose
   ```
2. Increase timeouts in your config file:
   ```yaml
   ai:
     timeout: 300  # 5 minutes instead of default 2 minutes
   ```

#### "Wrong programming language generated"
**What you'll see**: Asked for PHP but got JavaScript/React instead

**Solutions**:
1. Be extremely explicit in your request:
   ```bash
   # Instead of:
   ./orc create code "Create a hello world web page"
   
   # Use:
   ./orc create code "ONLY USE PHP. Create hello.php that echoes Hello World. No JavaScript, no React, ONLY PHP."
   ```
2. Include what you DON'T want: "No JavaScript", "No frameworks", etc.
3. Specify exact filenames: "Create hello.php" not just "Create hello world"

#### "JSON parsing error"
**What you'll see**: Errors mentioning "invalid character" or "unexpected token"

**Solution**: This is usually an internal issue that the orchestrator handles automatically. If it persists:
1. Try the request again - the system has retry logic
2. Use `--verbose` to see more details
3. Report the issue if it continues happening

### ⏱️ Timeout and Network Errors

#### "Request timed out"
**What you'll see**: Error message about timeout exceeded

**Solutions**:
1. For one-time fix, just retry the command
2. For persistent issues, increase timeouts in config:
   ```yaml
   ai:
     timeout: 300  # 5 minutes
   limits:
     rate_limit:
       requests_per_minute: 30  # Slower but more reliable
   ```
3. Use `--fluid` mode which has longer timeouts built-in

#### "Rate limit exceeded"
**What you'll see**: Error about too many requests

**Solutions**:
1. Wait a minute and try again - the system handles this automatically
2. Reduce rate limits in config if it happens frequently:
   ```yaml
   limits:
     rate_limit:
       requests_per_minute: 20  # Lower number = fewer errors
       burst_size: 3
   ```

### 💾 Output and Permission Errors

#### "Permission denied when saving output"
**What you'll see**: Errors when trying to write generated files

**Solutions**:
1. Check directory permissions:
   ```bash
   ls -la ~/.local/share/orchestrator/
   ```
2. Create directories if missing:
   ```bash
   mkdir -p ~/.local/share/orchestrator/output
   chmod 755 ~/.local/share/orchestrator/output
   ```
3. Make sure you own the directories:
   ```bash
   sudo chown -R $USER:$USER ~/.local/share/orchestrator/
   ```

#### "Disk space issues"
**What you'll see**: Errors about unable to write files

**Solutions**:
1. Check available space:
   ```bash
   df -h ~/.local/share/orchestrator/
   ```
2. Clean up old sessions:
   ```bash
   # List old sessions
   ls -la ~/.local/share/orchestrator/output/
   
   # Remove specific old sessions you don't need
   rm -rf ~/.local/share/orchestrator/output/old-session-id/
   ```

### 🔄 Resume and Session Errors

#### "Cannot resume session"
**What you'll see**: Error when trying to resume with `./orc resume SESSION_ID`

**Solutions**:
1. Check if the session exists:
   ```bash
   ls ~/.local/share/orchestrator/checkpoints/SESSION_ID*
   ```
2. Make sure you're using the same orchestrator type (if created with `--fluid`, resume with `--fluid`)
3. If incompatible, start a fresh session instead

#### "Session state corrupted"
**What you'll see**: Errors about invalid checkpoint data

**Solution**: Start a new session - corrupted checkpoints usually can't be recovered:
```bash
./orc create code "your request" --verbose
```

## Debugging Steps

### 1. Enable Verbose Output
Always use `--verbose` when troubleshooting:
```bash
./orc create code "your request" --verbose
```

### 2. Check the Debug Log
The debug log has detailed error information:
```bash
# View recent errors
tail -f ~/.local/state/orchestrator/debug.log

# Search for specific errors
grep "ERROR" ~/.local/state/orchestrator/debug.log
```

### 3. Verify Your Setup
Run these commands to check your installation:
```bash
# Check if orchestrator is in PATH
which orc

# Verify config exists
ls -la ~/.config/orchestrator/config.yaml

# Check API key is set
echo $OPENAI_API_KEY | cut -c1-10...  # Shows first 10 chars for security

# Test with a simple request
./orc create code "Create a hello world in Python" --verbose
```

## Best Practices to Avoid Issues

### For Code Generation
1. **Be explicit about languages**: Say "ONLY PHP" not just "PHP"
2. **Include file names**: "Create index.html" not just "Create a web page"
3. **List what you DON'T want**: "No frameworks, no JavaScript, no CSS"
4. **Use quality mode**: Add `--fluid` for better results

### For Configuration
1. **Start with defaults**: Use the example config and modify gradually
2. **Test changes**: After config changes, test with simple requests first
3. **Back up working configs**: `cp ~/.config/orchestrator/config.yaml ~/.config/orchestrator/config.yaml.backup`

### For Large Projects
1. **Break into smaller requests**: Instead of "Build a complete e-commerce site", start with "Create user authentication"
2. **Save your prompts**: Keep successful prompts for reuse
3. **Use sessions**: Take advantage of checkpoints for long-running generations

## Getting Help

### Quick Fixes
- **Most errors are temporary**: Try your command again
- **Quality issues**: Use `--fluid` flag
- **Language issues**: Be more explicit in your request

### When to Check Logs
Check logs when:
- The same error happens repeatedly
- You get unexpected results
- The program crashes

### Useful Commands for Troubleshooting
```bash
# View your configuration
cat ~/.config/orchestrator/config.yaml

# Check recent errors
tail -20 ~/.local/state/orchestrator/debug.log

# See what model you're using
grep "model:" ~/.config/orchestrator/config.yaml

# List recent output sessions
ls -lat ~/.local/share/orchestrator/output/

# Check disk space
df -h ~/.local/share/orchestrator/
```

## Error Prevention Checklist

Before running orchestrator:
- [ ] API key is set: `echo $OPENAI_API_KEY`
- [ ] Config file exists: `ls ~/.config/orchestrator/config.yaml`
- [ ] Output directory is writable: `touch ~/.local/share/orchestrator/test && rm ~/.local/share/orchestrator/test`
- [ ] You have enough disk space: `df -h ~/.local/share/orchestrator/`

For each request:
- [ ] Language is explicitly stated ("ONLY Python", "ONLY PHP")
- [ ] Unwanted options are excluded ("No frameworks", "No React")
- [ ] File names are specified when relevant ("Create server.js")
- [ ] Quality flag is used for complex tasks (`--fluid`)

---

**Remember**: Most issues are temporary and can be resolved by retrying. When in doubt, use `--verbose` to see what's happening and check the debug log for details.