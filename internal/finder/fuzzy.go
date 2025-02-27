package finder

import (
	"bufio"
	"fcopy/internal/config"
	"fcopy/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FuzzyMatch represents a potential path match with a similarity score
type FuzzyMatch struct {
	Path      string
	Name      string
	Score     int
	IsDir     bool
	Depth     int    // Directory depth from search root
	MatchType string // Full or partial match type
}

// ShouldIgnore checks if a path should be ignored during fuzzy search
func ShouldIgnore(path string, isDir bool, cfg *config.Config) bool {
	// Don't skip anything if --no-ignore flag is set
	if cfg.NoIgnore {
		return false
	}

	// Check if it's a hidden file/directory and we're not including hidden files
	fileName := filepath.Base(path)
	if !cfg.SearchHidden && len(fileName) > 1 && fileName[0] == '.' {
		return true
	}

	// Check if directory should be ignored
	if isDir {
		return config.IgnoreDirs[fileName]
	}

	// Check file extensions to ignore
	ext := filepath.Ext(fileName)
	if config.IgnoreExts[ext] {
		return true
	}

	// Check for specific filename patterns
	for pattern := range config.IgnoreExts {
		if strings.HasSuffix(fileName, pattern) {
			return true
		}
	}

	return false
}

// FuzzyFindPath attempts to find a file or directory based on an approximate name
func FuzzyFindPath(approximatePath string, cfg *config.Config) (string, bool) {
	// Get the directory to search in and the target name
	dir := "."
	targetName := approximatePath

	// If the path contains a directory separator, split it
	if strings.Contains(approximatePath, string(os.PathSeparator)) {
		dir = filepath.Dir(approximatePath)
		targetName = filepath.Base(approximatePath)

		// Make sure the directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// If the directory doesn't exist, search for it first
			resolvedDir, found := FuzzyFindPath(dir, cfg)
			if !found {
				fmt.Printf("Cannot find directory: %s\n", dir)
				return "", false
			}
			dir = resolvedDir
		}
	}

	// Find potential matches recursively
	matches := FindRecursiveMatches(dir, targetName, 0, cfg)

	if len(matches) == 0 {
		fmt.Printf("No matches found for '%s' anywhere in '%s'\n", targetName, dir)
		return "", false
	}

	// Sort matches by score first, then by depth
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score < matches[j].Score // Lower score (more similar) is better
		}
		return matches[i].Depth < matches[j].Depth // Lower depth (closer to search root) is better
	})

	// Limit the number of matches to display
	displayCount := len(matches)
	if displayCount > cfg.MaxMatches {
		displayCount = cfg.MaxMatches
	}

	// Check if we should auto-select the best match
	if cfg.AutoSelect && len(matches) > 0 {
		bestMatch := matches[0]
		// Only auto-select if the score is very good (threshold depends on name length)
		threshold := len(targetName) / 4
		if threshold < 2 {
			threshold = 2
		}

		if bestMatch.Score <= threshold {
			fmt.Printf("Auto-selected best match for '%s': %s\n", approximatePath, bestMatch.Path)
			return bestMatch.Path, true
		}
	}

	// Display matches to user
	fmt.Printf("'%s' not found. Did you mean:\n", approximatePath)
	for i := 0; i < displayCount; i++ {
		match := matches[i]
		fileType := "file"
		if match.IsDir {
			fileType = "dir "
		}
		fmt.Printf("[%d] %s (%s, score: %d, depth: %d)\n",
			i+1, match.Path, fileType, match.Score, match.Depth)
	}
	fmt.Printf("[0] None of these\n")

	// Get user selection
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter selection (0-", displayCount, "): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return "", false
		}

		input = strings.TrimSpace(input)
		var selection int
		_, err = fmt.Sscanf(input, "%d", &selection)

		if err != nil || selection < 0 || selection > displayCount {
			fmt.Println("Invalid selection. Please try again.")
			continue
		}

		if selection == 0 {
			return "", false
		}

		return matches[selection-1].Path, true
	}
}

// FindRecursiveMatches finds all potential matches for targetName in dir and its subdirectories
func FindRecursiveMatches(dir, targetName string, currentDepth int, cfg *config.Config) []FuzzyMatch {
	// Check if we've exceeded max search depth
	if currentDepth > cfg.SearchDepth {
		return nil
	}

	var matches []FuzzyMatch

	// Get all entries in the current directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		if cfg.Verbose {
			fmt.Printf("Error reading directory %s: %v\n", dir, err)
		}
		return nil
	}

	targetLower := strings.ToLower(targetName)

	// First, check for direct matches in this directory
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		// Skip if this path should be ignored
		if ShouldIgnore(path, entry.IsDir(), cfg) {
			continue
		}

		nameLower := strings.ToLower(name)

		// Exact match is best
		if nameLower == targetLower {
			matches = append(matches, FuzzyMatch{
				Path:      path,
				Name:      name,
				Score:     0, // Perfect match
				IsDir:     entry.IsDir(),
				Depth:     currentDepth,
				MatchType: "exact",
			})
			continue
		}

		// Check for substring match
		if strings.Contains(nameLower, targetLower) || strings.Contains(targetLower, nameLower) {
			// Calculate how close this substring match is
			scoreFactor := utils.Abs(len(nameLower) - len(targetLower))
			matches = append(matches, FuzzyMatch{
				Path:      path,
				Name:      name,
				Score:     1 + scoreFactor, // Good match but not exact
				IsDir:     entry.IsDir(),
				Depth:     currentDepth,
				MatchType: "substring",
			})
			continue
		}

		// Calculate Levenshtein distance for fuzzy match
		score := utils.CalculateSimilarity(nameLower, targetLower)

		// Add to matches if the similarity score is above a threshold
		threshold := len(targetName) * 2 / 3
		if threshold < 3 {
			threshold = 3
		}

		if score <= threshold {
			matches = append(matches, FuzzyMatch{
				Path:      path,
				Name:      name,
				Score:     score + 2, // Fuzzy match (less weight than substring)
				IsDir:     entry.IsDir(),
				Depth:     currentDepth,
				MatchType: "fuzzy",
			})
		}
	}

	// Now recursively check subdirectories
	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())

			// Skip ignored directories
			if ShouldIgnore(subdir, true, cfg) {
				continue
			}

			// Search recursively in this subdirectory
			subMatches := FindRecursiveMatches(subdir, targetName, currentDepth+1, cfg)
			matches = append(matches, subMatches...)
		}
	}

	return matches
}
