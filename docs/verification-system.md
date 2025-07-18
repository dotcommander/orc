# Quality Verification Features

Orc includes an intelligent verification system that ensures your generated content meets high quality standards. This guide explains how to use and customize these features.

## What Verification Does

The verification system automatically checks your generated content for:

### Fiction Verification
- **Complete story structure**: All planned chapters are present
- **Minimum word counts**: Each chapter meets length requirements  
- **Character consistency**: Characters remain consistent throughout
- **Plot completeness**: All plot threads are resolved
- **Style consistency**: Writing style remains uniform

### Code Verification
- **Syntax validity**: Code compiles/runs without errors
- **Complete implementation**: All planned features are present
- **Error handling**: Proper error cases are covered
- **Documentation**: Code includes appropriate comments
- **Best practices**: Follows language conventions

## How It Works

### Automatic Retries
When content doesn't meet quality standards:
1. Orc automatically retries generation (up to 3 times)
2. Each retry incorporates feedback from the previous attempt
3. If issues persist, they're documented for your review

### Issue Tracking
Failed verifications are saved in an `issues/` directory:
```
output/sessions/<session-id>/issues/
├── summary.md          # Human-readable summary
└── details.json        # Detailed failure information
```

## Using Verification

### Enable with Fluid Mode
The most comprehensive verification happens in fluid mode:
```bash
# Fiction with full verification
orc create fiction "Write a mystery novel" --fluid

# Code with quality checks  
orc create code "Build a REST API" --fluid --verbose
```

### Verification Levels

Configure how strict verification should be in your `config.yaml`:

```yaml
# Strict verification (highest quality)
verification:
  strict_mode: true
  max_retries: 5
  min_word_count: 1000
  require_all_checks: true

# Balanced verification (default)
verification:
  strict_mode: false
  max_retries: 3
  min_word_count: 500
  require_all_checks: false

# Fast generation (minimal verification)
verification:
  enabled: false
```

## Understanding Issues

When verification fails, check the issues directory:

### Summary File Example
```markdown
# Verification Issues - Session abc123

## Writing Phase Issues
- Chapter 3: Only 450 words (minimum: 1000)
- Chapter 7: Missing character dialogue
- Chapter 12: Inconsistent timeline

## Recommendations
1. Re-run with --fluid for automatic fixes
2. Increase phase timeout for longer chapters
3. Review character consistency guidelines
```

### Common Issues and Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| "Content too short" | Timeout or model limitations | Increase `phase_timeout` in config |
| "Missing elements" | Incomplete generation | Use `--fluid` mode for better results |
| "Inconsistent style" | Multiple generation passes | Enable style verification |
| "Code won't compile" | Syntax errors | Use language-specific verification |

## Customizing Verification

### Add Custom Rules
You can add your own verification rules through configuration:

```yaml
# config.yaml
verification:
  custom_rules:
    - name: "technical_accuracy"
      description: "Verify technical terms are used correctly"
      enabled: true
    - name: "brand_consistency"
      description: "Ensure brand guidelines are followed"
      enabled: true
```

### Skip Specific Checks
Disable checks that aren't relevant to your project:

```yaml
verification:
  skip_checks:
    - "word_count"      # Don't enforce minimum lengths
    - "style_check"     # Allow varied writing styles
```

## Best Practices

### For Fiction
1. **Use fluid mode** for best results: `--fluid`
2. **Set realistic word counts** in your prompts
3. **Review issue summaries** to understand patterns
4. **Increase timeouts** for complex scenes

### For Code
1. **Specify the language** clearly in your prompt
2. **Use verbose mode** to see verification progress
3. **Enable strict mode** for production code
4. **Review generated tests** for completeness

## Troubleshooting

### Verification Keeps Failing
- Increase timeouts: `phase_timeout: "45m"`
- Try a different model: `ai.model: "claude-3-opus-20240229"`
- Simplify your request into smaller parts

### Too Many False Positives
- Disable strict mode: `strict_mode: false`
- Skip irrelevant checks
- Adjust minimum thresholds

### Need More Control
- Use custom rules for domain-specific requirements
- Implement post-processing scripts
- Contact support for advanced verification options

## Next Steps

- Learn about [`performance.md`](performance.md) optimization
- See [`configuration.md`](configuration.md) for all settings
- Check [`examples/`](examples/) for verification in action