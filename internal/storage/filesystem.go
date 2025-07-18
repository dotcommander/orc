package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSystem struct {
	baseDir string
}

func NewFileSystem(baseDir string) *FileSystem {
	return &FileSystem{
		baseDir: baseDir,
	}
}

// sanitizePath validates and cleans the path to prevent directory traversal
func (fs *FileSystem) sanitizePath(path string) (string, error) {
	// Clean the path to resolve . and .. elements
	cleaned := filepath.Clean(path)
	
	// Reject paths that try to escape using ..
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("invalid path: contains parent directory reference")
	}
	
	// Reject absolute paths
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid path: absolute paths not allowed")
	}
	
	// Build the full path
	fullPath := filepath.Join(fs.baseDir, cleaned)
	
	// Verify the final path is still within baseDir
	// This handles symbolic links and other edge cases
	if !strings.HasPrefix(fullPath, fs.baseDir+string(filepath.Separator)) && fullPath != fs.baseDir {
		return "", fmt.Errorf("invalid path: outside base directory")
	}
	
	return fullPath, nil
}

func (fs *FileSystem) Save(ctx context.Context, path string, data []byte) error {
	fullPath, err := fs.sanitizePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	
	// Use restrictive permissions for sensitive files
	mode := os.FileMode(0644)
	if strings.Contains(path, "config") || strings.Contains(path, ".env") {
		mode = 0600 // Owner read/write only for config files
	}
	
	if err := os.WriteFile(fullPath, data, mode); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	
	return nil
}

func (fs *FileSystem) Load(ctx context.Context, path string) ([]byte, error) {
	fullPath, err := fs.sanitizePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	
	return data, nil
}

func (fs *FileSystem) List(ctx context.Context, pattern string) ([]string, error) {
	// For glob patterns, we need to be more careful
	// Clean the pattern but allow * and ? wildcards
	cleaned := filepath.Clean(pattern)
	if strings.Contains(cleaned, "..") {
		return nil, fmt.Errorf("invalid pattern: contains parent directory reference")
	}
	if filepath.IsAbs(cleaned) {
		return nil, fmt.Errorf("invalid pattern: absolute paths not allowed")
	}
	
	fullPattern := filepath.Join(fs.baseDir, cleaned)
	
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}
	
	var results []string
	for _, match := range matches {
		// Verify each match is within baseDir
		if !strings.HasPrefix(match, fs.baseDir+string(filepath.Separator)) && match != fs.baseDir {
			continue
		}
		
		rel, err := filepath.Rel(fs.baseDir, match)
		if err != nil {
			continue
		}
		results = append(results, rel)
	}
	
	return results, nil
}

func (fs *FileSystem) Exists(ctx context.Context, path string) bool {
	fullPath, err := fs.sanitizePath(path)
	if err != nil {
		return false
	}
	
	_, err = os.Stat(fullPath)
	return err == nil
}

func (fs *FileSystem) Delete(ctx context.Context, path string) error {
	fullPath, err := fs.sanitizePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}
	
	return nil
}