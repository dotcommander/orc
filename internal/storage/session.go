package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// SessionNamingStrategy defines how to name session output directories
type SessionNamingStrategy int

const (
	// SessionUUID uses the full UUID (default)
	SessionUUID SessionNamingStrategy = iota
	// SessionTimestamp uses timestamp + short ID
	SessionTimestamp
	// SessionDescriptive uses timestamp + sanitized request snippet
	SessionDescriptive
)

// CreateSessionPath creates a session-specific output path based on the naming strategy
func CreateSessionPath(baseDir, sessionID, request string, strategy SessionNamingStrategy) string {
	switch strategy {
	case SessionTimestamp:
		// Format: 2025-07-16_1530_82f06b15
		timestamp := time.Now().Format("2006-01-02_1504")
		shortID := sessionID[:8]
		return filepath.Join(baseDir, "sessions", fmt.Sprintf("%s_%s", timestamp, shortID))
		
	case SessionDescriptive:
		// Format: 2025-07-16_1530_j2-haplogroup-novelette_82f06b15
		timestamp := time.Now().Format("2006-01-02_1504")
		shortID := sessionID[:8]
		
		// Sanitize request for filename
		sanitized := sanitizeForFilename(request, 30)
		
		return filepath.Join(baseDir, "sessions", fmt.Sprintf("%s_%s_%s", timestamp, sanitized, shortID))
		
	default:
		// Default: use full session UUID
		return filepath.Join(baseDir, "sessions", sessionID)
	}
}

// sanitizeForFilename converts a string to a safe filename component
func sanitizeForFilename(s string, maxLen int) string {
	// Convert to lowercase and replace spaces with hyphens
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	
	// Remove or replace problematic characters
	replacements := map[string]string{
		"/":  "-",
		"\\": "-",
		":":  "-",
		"*":  "",
		"?":  "",
		"\"": "",
		"<":  "",
		">":  "",
		"|":  "",
		".":  "-",
		",":  "",
		"'":  "",
		"!":  "",
		"@":  "",
		"#":  "",
		"$":  "",
		"%":  "",
		"^":  "",
		"&":  "",
		"(":  "",
		")":  "",
		"[":  "",
		"]":  "",
		"{":  "",
		"}":  "",
		";":  "",
		"=":  "",
		"+":  "",
	}
	
	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	
	// Remove multiple consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	
	// Trim hyphens from start and end
	s = strings.Trim(s, "-")
	
	// Truncate to max length
	if len(s) > maxLen {
		s = s[:maxLen]
		// Ensure we don't end with a hyphen after truncation
		s = strings.TrimRight(s, "-")
	}
	
	// If empty after sanitization, use a default
	if s == "" {
		s = "output"
	}
	
	return s
}

// CreateSessionMetadata creates a metadata file for the session
func CreateSessionMetadata(outputDir, sessionID, request, pluginName string) []byte {
	metadata := fmt.Sprintf(`# Session Metadata

**Session ID**: %s
**Date**: %s
**Plugin**: %s
**Request**: %s

## Output Files

This directory contains all output from the AI novel generation session.
`, sessionID, time.Now().Format("2006-01-02 15:04:05"), pluginName, request)
	
	return []byte(metadata)
}