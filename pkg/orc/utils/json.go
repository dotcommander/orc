package utils

import (
	"encoding/json"
	"strings"
)

// CleanJSONResponse removes markdown code blocks and cleans JSON for parsing
func CleanJSONResponse(response string) string {
	// Remove markdown code blocks
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	
	// Find JSON boundaries
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	
	if start >= 0 && end > start {
		response = response[start : end+1]
	}
	
	// Clean common issues
	response = strings.TrimSpace(response)
	
	return response
}

// ParseJSONResponse parses a potentially messy AI JSON response
func ParseJSONResponse(response string, target interface{}) error {
	cleaned := CleanJSONResponse(response)
	return json.Unmarshal([]byte(cleaned), target)
}

// MustParseJSON parses JSON or returns a default value
func MustParseJSON(response string, target interface{}, defaultValue interface{}) error {
	err := ParseJSONResponse(response, target)
	if err != nil {
		// Try to use default value
		defaultJSON, _ := json.Marshal(defaultValue)
		return json.Unmarshal(defaultJSON, target)
	}
	return nil
}