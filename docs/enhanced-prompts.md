# Enhanced Prompts System (V2)

## Overview

The Orchestrator now supports enhanced V2 prompts that follow Anthropic's 2025 prompt engineering best practices. These prompts deliver significantly better quality outputs through sophisticated prompt engineering techniques.

## Key Improvements

### 1. System Prompts for Role Assignment
Each AI agent now has a detailed persona with specific expertise:
- **Elena Voss** - Senior Narrative Architect for fiction planning
- **Sarah Chen** - Award-winning novelist for writing
- **Michael Torres** - Veteran editor for editing
- **Marcus Chen** - Senior Software Architect for code planning
- **Dr. Lisa Park** - Code analysis expert
- **Alex Rivera** - Full-stack developer for implementation

### 2. XML Structure for Clarity
All prompts use XML tags to organize information:
```xml
<system>Role and expertise</system>
<instructions>Clear task description</instructions>
<examples>3-5 detailed examples</examples>
<thinking_process>How to approach the task</thinking_process>
<success_criteria>What makes a good output</success_criteria>
```

### 3. Multishot Prompting
Each prompt includes 3-5 comprehensive examples showing:
- Input scenarios
- Expected outputs
- Analysis and reasoning
- Common patterns and anti-patterns

### 4. Chain-of-Thought Reasoning
Prompts guide the AI through structured thinking:
- Problem analysis
- Solution consideration
- Quality verification
- Iterative improvement

### 5. Domain-Specific Expertise
Each prompt demonstrates deep domain knowledge:
- Fiction: Market awareness, genre conventions, reader psychology
- Code: Security-first design, clean architecture, performance optimization

## Usage

### Enable Enhanced Prompts (Default)
```bash
./orc create fiction "Write a thriller about AI consciousness"
./orc create code "Build a secure authentication system"
```

### Use Legacy Prompts
```bash
./orc create fiction "Write a story" --legacy-prompts
./orc create code "Create an API" --legacy-prompts
```

## Implementation Details

### AgentFactory Pattern
The system uses an `AgentFactory` to manage prompt versions:
```go
factory := agent.NewAgentFactory(client, promptsDir, useV2)
agent := factory.CreateFictionAgent("planning")
```

### Enhanced Plugins
- `EnhancedFictionPlugin` - Uses V2 prompts for all fiction phases
- `EnhancedCodePlugin` - Uses V2 prompts for all code phases

### Backward Compatibility
The system maintains full backward compatibility:
- Default: Enhanced V2 prompts
- `--legacy-prompts` flag: Original prompts
- No changes to existing workflows

## Quality Improvements Observed

### Fiction Generation
- **Character Development**: Deep psychological profiles with clear arcs
- **Plot Structure**: Professional three-act structures with precise pacing
- **Scene Writing**: Cinematic, sensory-rich prose with strong hooks
- **Editing**: Line-level improvements with market awareness

### Code Generation
- **Architecture**: Clean, scalable designs with security considerations
- **Implementation**: Production-ready code with error handling
- **Analysis**: Comprehensive reviews covering security, performance, and maintainability
- **Documentation**: Clear, professional documentation with examples

## Technical Architecture

### Prompt Files
```
prompts/
├── orchestrator_v2.txt    # Fiction planning
├── writer_v2.txt          # Scene writing
├── editor_v2.txt          # Fiction editing
├── critic_v2.txt          # Literary critique
├── architect_v2.txt       # Story architecture
├── code_planner_v2.txt    # Code planning
├── code_analyzer_v2.txt   # Code analysis
├── code_implementer_v2.txt # Code implementation
└── code_reviewer_v2.txt   # Code review
```

### Integration Points
1. **Agent Creation**: Factory pattern for version management
2. **Phase Execution**: Enhanced phases use V2 agents
3. **System Prompts**: Proper role assignment via API
4. **Response Quality**: Structured outputs with clear formatting

## Best Practices

### When to Use Enhanced Prompts
- Production content generation
- Complex creative projects
- Professional code development
- Quality-critical outputs

### When to Use Legacy Prompts
- Quick prototypes
- Backward compatibility testing
- Simpler requirements
- Performance testing

## Future Enhancements

1. **Prompt Versioning**: Track and manage multiple prompt versions
2. **A/B Testing**: Compare outputs between versions
3. **Custom Personas**: User-defined agent personalities
4. **Prompt Marketplace**: Share and discover effective prompts
5. **Performance Metrics**: Measure quality improvements quantitatively

## Conclusion

The enhanced prompts system represents a significant leap in AI-driven content generation quality. By following Anthropic's best practices and implementing sophisticated prompt engineering techniques, the Orchestrator now produces outputs that meet professional standards across both creative and technical domains.