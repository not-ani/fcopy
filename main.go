package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.design/x/clipboard"
)

// FileContent represents a file's name and content
type FileContent struct {
	Path    string
	Content string
}

var (
	maxFileSizeFlag = flag.Int64("max-size", 1024*1024, "Maximum file size in bytes")
	timeoutFlag     = flag.Duration("timeout", 30*time.Second, "Timeout for operation")
	workersFlag     = flag.Int("workers", 10, "Number of concurrent workers")
	verboseFlag     = flag.Bool("verbose", false, "Verbose output")
	debugFlag       = flag.Bool("debug", true, "Enable debug mode") // Default to true for debugging
	logFile         *os.File
	logger          *log.Logger
)

func main() {
	// Setup debug log file
	var err error
	logFile, err = os.Create("fcopy_debug.log")
	if err != nil {
		fmt.Printf("Warning: Could not create debug log file: %v\n", err)
	} else {
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), *timeoutFlag)
	defer cancel()

	paths := flag.Args()

	fileContents := make(chan FileContent, 100)
	var wg sync.WaitGroup
	var processedFiles atomic.Int64
	var errorCount atomic.Int64

	// Process each path in parallel
	for i, path := range paths {
		wg.Add(1)
		go func(p string, idx int) {
			defer wg.Done()
			// Remove quotes if present
			cleanPath := strings.Trim(p, "\"'")
			processPath(ctx, cleanPath, fileContents, &processedFiles, &errorCount)
		}(path, i)
	}

	// Close results channel when all processing is done
	go func() {
		wg.Wait()
		close(fileContents)
	}()

	// Show progress periodically
	if *verboseFlag {
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

	if *verboseFlag {
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

func processPath(
	ctx context.Context,
	path string,
	results chan<- FileContent,
	processed *atomic.Int64,
	errors *atomic.Int64,
) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error accessing %s: %v\n", path, err)
		errors.Add(1)
		return
	}

	if fileInfo.IsDir() {
		// Process directory recursively
		processDirectory(ctx, path, results, processed, errors)
	} else {
		// Process single file
		if err := processSingleFile(ctx, path, fileInfo, results); err != nil {
			errors.Add(1)
			if *verboseFlag {
				fmt.Printf("Error processing %s: %v\n", path, err)
			}
		} else {
			processed.Add(1)
		}
	}
}

func processSingleFile(
	ctx context.Context,
	path string,
	fileInfo os.FileInfo,
	results chan<- FileContent,
) error {
	// Skip files that are too large
	if fileInfo.Size() > *maxFileSizeFlag {
		return fmt.Errorf("file too large (size: %d bytes)", fileInfo.Size())
	}

	// Skip binary files by extension (simple heuristic)
	ext := strings.ToLower(filepath.Ext(path))
	skipExts := map[string]bool{
		".bin": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	}
	if skipExts[ext] {
		return fmt.Errorf("skipped binary file")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		select {
		case results <- FileContent{
			Path:    path,
			Content: string(content),
		}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func processDirectory(
	ctx context.Context,
	dirPath string,
	results chan<- FileContent,
	processed *atomic.Int64,
	errors *atomic.Int64,
) {
	var wg sync.WaitGroup
	files := make(chan string, 100)

	// Start worker pool for processing files
	for i := 0; i < *workersFlag; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()
			for path := range files {
				fileInfo, err := os.Stat(path)
				if err != nil {
					if *verboseFlag {
						fmt.Printf("Error stating %s: %v\n", path, err)
					}
					errors.Add(1)
					continue
				}

				if err := processSingleFile(ctx, path, fileInfo, results); err != nil {
					errors.Add(1)
					if *verboseFlag && err != context.Canceled {
						fmt.Printf("Error processing %s: %v\n", path, err)
					}
				} else {
					processed.Add(1)
				}
			}
		}(i)
	}

	// Walk directory and send files to worker pool
	fileCount := 0
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			fileCount++
			select {
			case files <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	close(files)

	if err != nil && err != context.Canceled {
		fmt.Printf("Error walking directory %s: %v\n", dirPath, err)
		errors.Add(1)
	}

	wg.Wait()
}
