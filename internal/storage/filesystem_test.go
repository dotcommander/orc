package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemSecurity(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "orc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test file outside the base directory
	outsideFile := filepath.Join(filepath.Dir(tempDir), "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outsideFile)
	
	fs := NewFileSystem(tempDir)
	ctx := context.Background()
	
	t.Run("Save prevents directory traversal", func(t *testing.T) {
		tests := []struct {
			name string
			path string
			want bool // true if should succeed
		}{
			{"normal path", "test.txt", true},
			{"subdirectory", "subdir/test.txt", true},
			{"parent traversal", "../test.txt", false},
			{"complex traversal", "subdir/../../test.txt", false},
			{"absolute path", "/etc/passwd", false},
			{"hidden traversal", "subdir/../../../etc/passwd", false},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := fs.Save(ctx, tt.path, []byte("test"))
				if tt.want && err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if !tt.want && err == nil {
					t.Errorf("expected error for path %q, got none", tt.path)
				}
			})
		}
	})
	
	t.Run("Load prevents directory traversal", func(t *testing.T) {
		// Create a valid test file
		validPath := filepath.Join(tempDir, "valid.txt")
		if err := os.WriteFile(validPath, []byte("valid"), 0644); err != nil {
			t.Fatal(err)
		}
		
		tests := []struct {
			name string
			path string
			want bool
		}{
			{"normal path", "valid.txt", true},
			{"parent traversal", "../outside.txt", false},
			{"absolute path", outsideFile, false},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := fs.Load(ctx, tt.path)
				if tt.want && err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if !tt.want && err == nil {
					t.Errorf("expected error for path %q, got none", tt.path)
				}
			})
		}
	})
	
	t.Run("List prevents directory traversal", func(t *testing.T) {
		tests := []struct {
			name    string
			pattern string
			want    bool
		}{
			{"normal pattern", "*.txt", true},
			{"subdirectory pattern", "subdir/*.txt", true},
			{"parent traversal", "../*", false},
			{"absolute pattern", "/etc/*", false},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := fs.List(ctx, tt.pattern)
				if tt.want && err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if !tt.want && err == nil {
					t.Errorf("expected error for pattern %q, got none", tt.pattern)
				}
			})
		}
	})
}

func TestSanitizePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "orc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	fs := &FileSystem{baseDir: tempDir}
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"simple file", "file.txt", false},
		{"nested file", "dir/file.txt", false},
		{"dot file", ".hidden", false},
		{"parent directory", "../file.txt", true},
		{"sneaky parent", "dir/../../../etc/passwd", true},
		{"absolute path", "/etc/passwd", true},
		{"empty path", "", false},
		{"dot path", ".", false},
		{"double dot", "..", true},
		{"contains double dot", "some/..thing/file", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fs.sanitizePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("sanitizePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if err == nil && !filepath.HasPrefix(got, tempDir) {
				t.Errorf("sanitizePath(%q) = %q, not under base directory %q", tt.path, got, tempDir)
			}
		})
	}
}