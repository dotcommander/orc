# Enhanced Prompts System (V2) - Default

## Overview

The Orchestrator uses enhanced V2 prompts that follow Anthropic's 2025 prompt engineering best practices. These are now the default and only prompts used by the system, delivering consistently high-quality outputs through sophisticated prompt engineering techniques.

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

All commands automatically use the enhanced V2 prompts:

```bash
# Fiction generation
./orc create fiction "Write a thriller about AI consciousness"
./orc create fiction "Create a mystery novel set in Victorian London"

# Code generation
./orc create code "Build a secure authentication system"
./orc create code "Create a REST API with JWT authentication"
```

## Implementation Details

### AgentFactory Pattern
The system uses an `AgentFactory` to create agents with V2 prompts:
```go
factory := agent.NewAgentFactory(client, promptsDir)
agent := factory.CreateFictionAgent("planning")
```

### Standard Plugins
- `FictionPlugin` - Uses V2 prompts for all fiction phases
- `CodePlugin` - Uses V2 prompts for all code phases

### Quality-First Architecture
The system is built around quality:
- All agents use enhanced V2 prompts
- Professional-grade outputs
- Consistent high quality across all domains

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

### Quality-First Development
The enhanced V2 prompts ensure:
- Production-ready content generation
- Professional handling of complex projects
- Enterprise-grade code development
- Consistently high-quality outputs

### Prompt Engineering Excellence
- Clear XML structure for organization
- Multi-shot examples for context
- Chain-of-thought reasoning
- Domain-specific expertise

## Future Enhancements

1. **Prompt Versioning**: Track and manage multiple prompt versions
2. **A/B Testing**: Compare outputs between versions
3. **Custom Personas**: User-defined agent personalities
4. **Prompt Marketplace**: Share and discover effective prompts
5. **Performance Metrics**: Measure quality improvements quantitatively

## Conclusion

The enhanced prompts system is the cornerstone of the Orchestrator's quality-first approach. By following Anthropic's best practices and implementing sophisticated prompt engineering techniques, the Orchestrator consistently produces outputs that meet professional standards across both creative and technical domains. This commitment to quality through advanced prompt engineering ensures that every generation delivers value.