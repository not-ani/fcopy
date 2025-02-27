package main

import (
	"context"
	"fcopy/internal/config"
	"fcopy/internal/finder"
	"fcopy/internal/processor"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.design/x/clipboard"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Warning: Could not create debug log file: %v\n", err)
	}
	if cfg.LogFile != nil {
		defer cfg.LogFile.Close()
	}

	// Parse flags
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("Usage: fcopy [options] <file1.ts> <folder/> ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	err = clipboard.Init()
	if err != nil {
		fmt.Printf("Failed to initialize clipboard: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	paths := flag.Args()
	resolvedPaths := make([]string, 0, len(paths))

	// First, resolve all paths with fuzzy matching if needed
	for _, path := range paths {
		// Remove quotes if present
		cleanPath := strings.Trim(path, "\"'")

		// Check if path exists
		if _, err := os.Stat(cleanPath); err != nil {
			if os.IsNotExist(err) {
				// Path doesn't exist, try fuzzy matching
				resolvedPath, found := finder.FuzzyFindPath(cleanPath, cfg)
				if found {
					resolvedPaths = append(resolvedPaths, resolvedPath)
				} else {
					fmt.Printf("Warning: Skipping %s as no good match was found\n", cleanPath)
				}
			} else {
				fmt.Printf("Error accessing %s: %v\n", cleanPath, err)
			}
		} else {
			// Path exists, use it as-is
			resolvedPaths = append(resolvedPaths, cleanPath)
		}
	}

	if len(resolvedPaths) == 0 {
		fmt.Println("No valid paths to process.")
		os.Exit(1)
	}

	fileContents := make(chan processor.FileContent, 100)
	var wg sync.WaitGroup
	var processedFiles atomic.Int64
	var errorCount atomic.Int64

	// Process each resolved path in parallel
	for i, path := range resolvedPaths {
		wg.Add(1)
		go func(p string, idx int) {
			defer wg.Done()
			processor.ProcessPath(ctx, p, cfg, fileContents, &processedFiles, &errorCount)
		}(path, i)
	}

	// Close results channel when all processing is done
	go func() {
		wg.Wait()
		close(fileContents)
	}()

	// Show progress periodically
	if cfg.Verbose {
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					fmt.Printf("\rProcessed: %d files", processedFiles.Load())
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Collect results
	var output strings.Builder
	count := 0
	for result := range fileContents {
		count++
		output.WriteString(fmt.Sprintf("-- %s --\n", result.Path))
		output.WriteString(result.Content)
		output.WriteString("\n\n")
	}

	if cfg.Verbose {
		fmt.Println() // New line after progress indicator
	}

	// Verify we have content to copy
	if output.Len() == 0 {
		fmt.Println("No content was found to copy!")
	} else {
		// Copy to clipboard
		data := []byte(output.String())
		clipboard.Write(clipboard.FmtText, data)

		fmt.Printf("Copied content from %d files to clipboard (%d bytes)\n",
			count, output.Len())
	}

	if errors := errorCount.Load(); errors > 0 {
		fmt.Printf(" (%d errors occurred)\n", errors)
	}
}
