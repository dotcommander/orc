# Example Orc Plugin

This is a minimal example plugin demonstrating how to create plugins for Orc.

## Features

- Simple two-phase pipeline (Analysis → Generation)
- No external dependencies
- Clear code structure
- Can be built as .so or binary

## Building

```bash
make build
```

## What It Does

1. **Analysis Phase**: Analyzes the user request (word count, timestamp)
2. **Generation Phase**: Creates a simple text output using the analysis

## Code Structure

- `plugin.go` - Main plugin implementation
- Uses the plugin SDK for base functionality
- Implements two simple phases
- Shows how to pass data between phases

## Key Concepts Demonstrated

1. **Plugin Structure**: How to structure a plugin
2. **Phase Implementation**: Creating custom phases
3. **Data Flow**: Passing data between phases
4. **SDK Usage**: Using the BasePlugin and BasePhase

## Usage

This plugin responds to any request with a simple analysis:

```
orc create example "Analyze this text"
```

Output:
```
Example Plugin Output

Request: Analyze this text
Analyzed at: 2024-01-20T10:30:00Z
Word count: 3
```

## Extending

To create your own plugin:
1. Copy this example
2. Rename the package and types
3. Implement your own phases
4. Add AI integration if needed
5. Update the manifest