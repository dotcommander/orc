package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// MockClient provides fake AI responses for testing
type MockClient struct {
	responses map[string]string
}

// NewMockClient creates a mock AI client for testing
func NewMockClient() *MockClient {
	return &MockClient{
		responses: map[string]string{
			"analyzer": `{
				"language": "Python",
				"framework": "",
				"complexity": "Simple",
				"main_objective": "Create a basic calculator with arithmetic operations",
				"requirements": [
					"Support addition, subtraction, multiplication, division",
					"Handle user input",
					"Display results"
				],
				"constraints": [
					"Keep it simple and beginner-friendly"
				],
				"potential_risks": [
					"Division by zero handling"
				]
			}`,
			"planner": `{
				"overview": "Build a simple command-line calculator in Python",
				"steps": [
					{
						"order": 1,
						"description": "Create main calculator module",
						"code_files": ["calculator.py"],
						"rationale": "Core calculator logic",
						"time_estimate": "15 minutes"
					},
					{
						"order": 2,
						"description": "Add input handling",
						"code_files": ["calculator.py"],
						"rationale": "User interaction",
						"time_estimate": "10 minutes"
					}
				],
				"testing": {
					"unit_tests": ["Test arithmetic operations", "Test error handling"],
					"integration_tests": ["Test full calculator flow"],
					"edge_cases": ["Division by zero", "Invalid input"]
				}
			}`,
			"implementer": `{
				"files": [
					{
						"path": "calculator.py",
						"content": "def add(a, b):\n    return a + b\n\ndef subtract(a, b):\n    return a - b\n\ndef multiply(a, b):\n    return a * b\n\ndef divide(a, b):\n    if b == 0:\n        raise ValueError('Cannot divide by zero')\n    return a / b\n\nif __name__ == '__main__':\n    print('Simple Calculator')\n    # Main calculator loop here",
						"language": "python",
						"purpose": "Main calculator implementation"
					}
				],
				"summary": "Basic calculator with four operations",
				"run_instructions": "Run with: python calculator.py"
			}`,
			"reviewer": `{
				"score": 8.5,
				"summary": "Clean, simple implementation suitable for beginners",
				"strengths": [
					"Clear function names",
					"Proper error handling for division by zero",
					"Simple and readable code"
				],
				"improvements": [
					{
						"priority": "Medium",
						"description": "Add input validation",
						"location": "Main block",
						"suggestion": "Validate user input before operations"
					}
				],
				"security_issues": [],
				"best_practices": ["Good function separation", "Error handling present"]
			}`,
		},
	}
}

// Complete returns a mock response
func (m *MockClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Detect which phase based on prompt content
	promptLower := strings.ToLower(prompt)
	
	if strings.Contains(promptLower, "analyze") || strings.Contains(promptLower, "analysis") {
		return m.responses["analyzer"], nil
	}
	if strings.Contains(promptLower, "plan") || strings.Contains(promptLower, "implementation plan") {
		return m.responses["planner"], nil
	}
	if strings.Contains(promptLower, "implement") || strings.Contains(promptLower, "code") {
		return m.responses["implementer"], nil
	}
	if strings.Contains(promptLower, "review") || strings.Contains(promptLower, "critique") {
		return m.responses["reviewer"], nil
	}
	
	// Default response
	return `{"message": "Mock response"}`, nil
}

// CompleteJSON returns a mock JSON response
func (m *MockClient) CompleteJSON(ctx context.Context, prompt string) (string, error) {
	response, err := m.Complete(ctx, prompt)
	if err != nil {
		return "", err
	}
	
	// Validate it's proper JSON
	var test interface{}
	if err := json.Unmarshal([]byte(response), &test); err != nil {
		return "", fmt.Errorf("mock response is not valid JSON: %w", err)
	}
	
	return response, nil
}