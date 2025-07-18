# Code Examples

**AI Context**: Practical code examples and usage patterns for The Orchestrator components. Use these as templates for implementing features or understanding best practices.

**Cross-references**: [`../api.md`](../api.md) for interface documentation, [`../development.md`](../development.md) for contribution guidelines.

## Quick Reference

| Example | Purpose | Complexity |
|---------|---------|------------|
| [`basic-usage.go`](basic-usage.go) | Simple orchestrator setup | Beginner |
| [`custom-phase.go`](custom-phase.go) | Implementing a new phase | Intermediate |
| [`mock-testing.go`](mock-testing.go) | Testing with mocks | Intermediate |
| [`advanced-agent.go`](advanced-agent.go) | Custom AI agent implementation | Advanced |
| [`storage-backends.go`](storage-backends.go) | Alternative storage options | Advanced |
| [`integration-test.go`](integration-test.go) | Full pipeline testing | Advanced |

## File Structure

```
examples/
├── README.md              # This file
├── basic-usage.go         # Simple orchestrator usage
├── custom-phase.go        # New phase implementation
├── mock-testing.go        # Testing patterns
├── advanced-agent.go      # Custom agent features
├── storage-backends.go    # Storage implementations
├── integration-test.go    # End-to-end testing
└── cli-examples/          # Command-line usage examples
    ├── basic-commands.md
    ├── configuration.md
    └── troubleshooting.md
```

## Usage Guidelines

### For AI Assistants
1. **Copy and adapt** - Use examples as starting points
2. **Understand patterns** - Each example demonstrates specific design patterns
3. **Check dependencies** - Ensure all imports are available
4. **Test thoroughly** - All examples include test patterns

### For Developers
1. **Run examples** - All code is functional and tested
2. **Modify safely** - Examples use interfaces for easy customization
3. **Learn gradually** - Start with basic examples, progress to advanced
4. **Contribute back** - Add new examples for common use cases

---

**Note**: All examples follow the project's coding standards and architectural patterns. See [`../development.md`](../development.md) for detailed guidelines.