package processor

import (
	"context"
	"fcopy/internal/config"
	"fcopy/internal/finder"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// FileContent represents a file's name and content
type FileContent struct {
	Path    string
	Content string
}

// ProcessPath processes a single path which may be a file or directory
func ProcessPath(
	ctx context.Context,
	path string,
	cfg *config.Config,
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
		ProcessDirectory(ctx, path, cfg, results, processed, errors)
	} else {
		// Process single file
		if err := ProcessSingleFile(ctx, path, fileInfo, cfg, results); err != nil {
			errors.Add(1)
			if cfg.Verbose {
				fmt.Printf("Error processing %s: %v\n", path, err)
			}
		} else {
			processed.Add(1)
		}
	}
}

// ProcessSingleFile processes a single file
func ProcessSingleFile(
	ctx context.Context,
	path string,
	fileInfo os.FileInfo,
	cfg *config.Config,
	results chan<- FileContent,
) error {
	// Skip files that are too large
	if fileInfo.Size() > cfg.MaxFileSize {
		return fmt.Errorf("file too large (size: %d bytes)", fileInfo.Size())
	}

	// Skip binary files by extension (simple heuristic)
	ext := strings.ToLower(filepath.Ext(path))
	if config.BinaryExts[ext] {
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

// ProcessDirectory processes a directory recursively
func ProcessDirectory(
	ctx context.Context,
	dirPath string,
	cfg *config.Config,
	results chan<- FileContent,
	processed *atomic.Int64,
	errors *atomic.Int64,
) {
	var wg sync.WaitGroup
	files := make(chan string, 100)

	// Start worker pool for processing files
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()
			for path := range files {
				fileInfo, err := os.Stat(path)
				if err != nil {
					if cfg.Verbose {
						fmt.Printf("Error stating %s: %v\n", path, err)
					}
					errors.Add(1)
					continue
				}

				if err := ProcessSingleFile(ctx, path, fileInfo, cfg, results); err != nil {
					errors.Add(1)
					if cfg.Verbose && err != context.Canceled {
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

		// Skip ignored directories
		if d.IsDir() && finder.ShouldIgnore(path, true, cfg) {
			return filepath.SkipDir
		}

		if !d.IsDir() {
			// Skip ignored files
			if finder.ShouldIgnore(path, false, cfg) {
				return nil
			}

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
