package phase

import (
	"encoding/json"
	"regexp"
	"strings"
)

// CleanJSONResponse removes markdown code blocks from AI responses and fixes common JSON issues.
// This handles responses that come wrapped in ```json ... ``` or just ``` ... ```
func CleanJSONResponse(response string) string {
	// Trim whitespace
	response = strings.TrimSpace(response)
	
	// Check if response is wrapped in markdown code blocks
	if strings.HasPrefix(response, "```json") && strings.HasSuffix(response, "```") {
		// Remove opening ```json
		response = strings.TrimPrefix(response, "```json")
		// Remove closing ```
		response = strings.TrimSuffix(response, "```")
		// Trim any whitespace left
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") && strings.HasSuffix(response, "```") {
		// Handle case where it's just ``` without json specification
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}
	
	// Try to extract JSON from the response if it contains non-JSON content
	response = extractJSON(response)
	
	return response
}

// extractJSON attempts to find and extract valid JSON from a response that may contain other text
func extractJSON(response string) string {
	// First try to parse as-is
	if isValidJSON(response) {
		return response
	}
	
	// Look for JSON-like structures starting with { and ending with }
	start := strings.Index(response, "{")
	if start == -1 {
		return response // No JSON found, return as-is
	}
	
	// Find the matching closing brace
	braceCount := 0
	var end int
	for i := start; i < len(response); i++ {
		if response[i] == '{' {
			braceCount++
		} else if response[i] == '}' {
			braceCount--
			if braceCount == 0 {
				end = i + 1
				break
			}
		}
	}
	
	if end == 0 {
		return response // No matching brace found
	}
	
	jsonCandidate := response[start:end]
	
	// Try to fix common JSON issues
	jsonCandidate = fixJSONString(jsonCandidate)
	
	if isValidJSON(jsonCandidate) {
		return jsonCandidate
	}
	
	return response // Return original if we can't fix it
}

// fixJSONString attempts to fix common JSON string issues
func fixJSONString(jsonStr string) string {
	// Fix unescaped newlines in string values
	// This regex finds strings and replaces literal newlines with \n
	re := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)`)
	
	jsonStr = re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		// Don't modify the opening quote
		if len(match) <= 1 {
			return match
		}
		
		content := match[1:] // Remove opening quote
		// Replace literal newlines with escaped newlines
		content = strings.ReplaceAll(content, "\n", "\\n")
		content = strings.ReplaceAll(content, "\r", "\\r")
		content = strings.ReplaceAll(content, "\t", "\\t")
		
		return `"` + content
	})
	
	// Fix common trailing comma issues
	jsonStr = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(jsonStr, "$1")
	
	// Fix missing quotes around object keys
	jsonStr = regexp.MustCompile(`([{,]\s*)([a-zA-Z_][a-zA-Z0-9_]*)\s*:`).ReplaceAllString(jsonStr, `$1"$2":`)
	
	// Fix unescaped quotes inside string values (basic attempt)
	jsonStr = regexp.MustCompile(`"([^"]*)"([^":,}\]]*)"([^"]*)":`).ReplaceAllString(jsonStr, `"$1\"$2\"$3":`)
	
	return jsonStr
}

// isValidJSON checks if a string is valid JSON
func isValidJSON(str string) bool {
	var js interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}