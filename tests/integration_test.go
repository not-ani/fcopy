package tests

import (
	"context"
	"fcopy/internal/config"
	"fcopy/internal/finder"
	"fcopy/internal/processor"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegration tests the main components working together
func TestIntegration(t *testing.T) {
	// Create a temporary test directory
	tempDir, err := ioutil.TempDir("", "integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directory structure
	files := []struct {
		path    string
		content string
	}{
		{"config.json", `{"setting": "value"}`},
		{"configuration.yaml", "setting: value"},
		{"docs/readme.md", "# Documentation"},
		{"src/main.go", "package main\n\nfunc main() {}"},
		{"test/test_file.go", "package test\n\nfunc TestFunc() {}"},
		{".gitignore", "node_modules/\ndist/"},
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file.path)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := ioutil.WriteFile(path, []byte(file.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Create test config
	cfg := &config.Config{
		MaxFileSize:  1024 * 1024,
		Timeout:      5 * time.Second,
		Workers:      2,
		Verbose:      true,
		Debug:        true,
		MaxMatches:   5,
		SearchDepth:  3,
		AutoSelect:   true, // Enable auto-select for testing
		SearchHidden: true,
		NoIgnore:     false,
	}

	// Test fuzzy finding
	t.Run("Fuzzy File Finding", func(t *testing.T) {
		// Change to the temp directory for this test
		originalDir, _ := os.Getwd()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}
		defer os.Chdir(originalDir)

		testCases := []struct {
			input       string
			expectedEnd string // Just check if the result ends with this
		}{
			{"config", "config.json"},          // Should match config.json (closest)
			{"readme", "docs/readme.md"},       // Should find in subdirectory
			{"main", "src/main.go"},            // Should find in src directory
			{"test_file", "test/test_file.go"}, // Should find in test directory
		}

		for _, tc := range testCases {
			// The FuzzyFindPath function interacts with stdin, so we'll need to modify
			// it for testing or test a different function that it uses

			// We'll test FindRecursiveMatches which is used by FuzzyFindPath
			matches := finder.FindRecursiveMatches(".", tc.input, 0, cfg)

			if len(matches) == 0 {
				t.Errorf("No matches found for '%s'", tc.input)
				continue
			}

			// Check if the best match (first one) ends with the expected string
			if !strings.HasSuffix(matches[0].Path, tc.expectedEnd) {
				t.Errorf("Expected best match for '%s' to end with '%s', got '%s'",
					tc.input, tc.expectedEnd, matches[0].Path)
			}
		}
	})

	// Test file processing
	t.Run("File Processing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results := make(chan processor.FileContent, 10)
		processed := &atomic.Int64{}
		errors := &atomic.Int64{}

		// Process the entire directory
		go processor.ProcessDirectory(ctx, tempDir, cfg, results, processed, errors)

		// Count and verify results
		foundFiles := make(map[string]bool)
		expectedCount := len(files)

		for i := 0; i < expectedCount; i++ {
			select {
			case result := <-results:
				// Get relative path
				relPath, err := filepath.Rel(tempDir, result.Path)
				if err != nil {
					t.Errorf("Failed to get relative path: %v", err)
					continue
				}

				// Check if this is one of our test files
				for _, testFile := range files {
					if testFile.path == relPath {
						foundFiles[relPath] = true
						// Verify content
						if result.Content != testFile.content {
							t.Errorf("Content mismatch for %s", relPath)
						}
						break
					}
				}
			case <-time.After(2 * time.Second):
				t.Errorf("Timed out waiting for results, got %d of %d files",
					len(foundFiles), expectedCount)
				break
			}
		}

		// Ensure we found all expected files
		for _, file := range files {
			if !foundFiles[file.path] {
				t.Errorf("File %s was not processed", file.path)
			}
		}

		// Check final counters
		if processed.Load() != int64(expectedCount) {
			t.Errorf("Expected %d processed files, got %d", expectedCount, processed.Load())
		}
		if errors.Load() != 0 {
			t.Errorf("Expected 0 errors, got %d", errors.Load())
		}
	})
}
