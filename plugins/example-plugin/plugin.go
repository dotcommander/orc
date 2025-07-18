package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dotcommander/orc/pkg/orc"
	"github.com/dotcommander/orc/pkg/plugin-sdk"
)

// ExamplePlugin demonstrates a simple Orc plugin
type ExamplePlugin struct {
	sdk.BasePlugin
}

// NewPlugin creates the example plugin
func NewPlugin() *ExamplePlugin {
	p := &ExamplePlugin{}
	p.BasePlugin = sdk.NewBasePlugin(
		"example",
		"1.0.0",
		"Example plugin demonstrating Orc plugin development",
		"Orc Community",
		[]string{"example", "demo"},
	)
	
	// Set our phases
	p.SetPhases([]orc.Phase{
		&AnalysisPhase{},
		&GenerationPhase{},
	})
	
	return p
}

// GetOutputSpec describes what this plugin produces
func (p *ExamplePlugin) GetOutputSpec() orc.OutputSpec {
	return orc.OutputSpec{
		PrimaryOutput:    "output.txt",
		SecondaryOutputs: []string{"summary.md", "metadata.json"},
		FilePatterns: map[string]string{
			"output": "*.txt",
			"docs":   "*.md",
		},
	}
}

// AnalysisPhase analyzes the user request
type AnalysisPhase struct {
	sdk.BasePhase
}

func NewAnalysisPhase() *AnalysisPhase {
	return &AnalysisPhase{
		BasePhase: sdk.NewBasePhase("Analysis", 30*time.Second),
	}
}

func (p *AnalysisPhase) Execute(ctx context.Context, input orc.PhaseInput) (orc.PhaseOutput, error) {
	// Simple analysis - just extract key words
	analysis := map[string]interface{}{
		"request":   input.Request,
		"wordCount": len(input.Request),
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	return orc.PhaseOutput{
		Data: analysis,
		Metadata: map[string]interface{}{
			"phase": "analysis",
		},
	}, nil
}

// GenerationPhase generates the output
type GenerationPhase struct {
	sdk.BasePhase
}

func NewGenerationPhase() *GenerationPhase {
	return &GenerationPhase{
		BasePhase: sdk.NewBasePhase("Generation", 1*time.Minute),
	}
}

func (p *GenerationPhase) Execute(ctx context.Context, input orc.PhaseInput) (orc.PhaseOutput, error) {
	// Get analysis from previous phase
	analysis, ok := input.PreviousOutputs["Analysis"].(map[string]interface{})
	if !ok {
		return orc.PhaseOutput{}, fmt.Errorf("missing analysis data")
	}
	
	// Generate simple output
	output := fmt.Sprintf(
		"Example Plugin Output\n\nRequest: %s\nAnalyzed at: %s\nWord count: %v\n",
		analysis["request"],
		analysis["timestamp"],
		analysis["wordCount"],
	)
	
	return orc.PhaseOutput{
		Data: output,
		Metadata: map[string]interface{}{
			"phase":  "generation",
			"length": len(output),
		},
	}, nil
}

// For plugin loading
var Plugin = NewPlugin()

// For binary mode
func main() {
	// This would start a JSON-RPC server for binary plugin mode
	fmt.Println("Example plugin would start in binary mode here")
}