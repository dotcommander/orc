package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptCache(t *testing.T) {
	// Create a temporary directory and file
	tempDir, err := os.MkdirTemp("", "prompt-cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "This is a test prompt template with {{.variable}}"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	cache := NewPromptCache()
	
	t.Run("loads prompt from file", func(t *testing.T) {
		content, err := cache.LoadPrompt(testFile)
		if err != nil {
			t.Fatalf("LoadPrompt() error = %v", err)
		}
		
		if content != testContent {
			t.Errorf("LoadPrompt() = %q, want %q", content, testContent)
		}
	})
	
	t.Run("caches prompt content", func(t *testing.T) {
		// First load
		_, err := cache.LoadPrompt(testFile)
		if err != nil {
			t.Fatal(err)
		}
		
		// Modify file after first load
		newContent := "Modified content"
		if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
			t.Fatal(err)
		}
		
		// Second load should return cached content, not new content
		content, err := cache.LoadPrompt(testFile)
		if err != nil {
			t.Fatal(err)
		}
		
		if content != testContent {
			t.Errorf("LoadPrompt() = %q, want cached content %q", content, testContent)
		}
	})
	
	t.Run("loads and caches template", func(t *testing.T) {
		tmpl, err := cache.LoadTemplate("test", testFile)
		if err != nil {
			t.Fatalf("LoadTemplate() error = %v", err)
		}
		
		if tmpl.Name() != "test" {
			t.Errorf("template name = %q, want %q", tmpl.Name(), "test")
		}
	})
	
	t.Run("preload multiple files", func(t *testing.T) {
		// Create another test file
		testFile2 := filepath.Join(tempDir, "test2.txt")
		testContent2 := "Second test prompt"
		if err := os.WriteFile(testFile2, []byte(testContent2), 0644); err != nil {
			t.Fatal(err)
		}
		
		newCache := NewPromptCache()
		paths := []string{testFile, testFile2}
		
		err := newCache.Preload(paths)
		if err != nil {
			t.Fatalf("Preload() error = %v", err)
		}
		
		// Check that both are cached
		_, raw := newCache.Stats()
		if raw != 2 {
			t.Errorf("Stats() raw = %d, want 2", raw)
		}
	})
	
	t.Run("clear cache", func(t *testing.T) {
		cache.Clear()
		templates, raw := cache.Stats()
		
		if templates != 0 || raw != 0 {
			t.Errorf("Stats() after Clear() = (%d, %d), want (0, 0)", templates, raw)
		}
	})
	
	t.Run("handles missing file", func(t *testing.T) {
		_, err := cache.LoadPrompt("nonexistent.txt")
		if err == nil {
			t.Error("LoadPrompt() with nonexistent file should return error")
		}
	})
}