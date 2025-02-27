package config

import (
	"flag"
	"log"
	"os"
	"time"
)

// Config holds the application configuration
type Config struct {
	MaxFileSize  int64
	Timeout      time.Duration
	Workers      int
	Verbose      bool
	Debug        bool
	MaxMatches   int
	SearchDepth  int
	AutoSelect   bool
	SearchHidden bool
	NoIgnore     bool
	Logger       *log.Logger
	LogFile      *os.File
}

// IgnoreDirs contains directories to skip during search
var IgnoreDirs = map[string]bool{
	"node_modules":     true,
	".git":             true,
	".svn":             true,
	".hg":              true,
	"dist":             true,
	"build":            true,
	"out":              true,
	"target":           true,
	"bin":              true,
	"obj":              true,
	".idea":            true,
	".vscode":          true,
	".vs":              true,
	"vendor":           true,
	"bower_components": true,
	"jspm_packages":    true,
	"tmp":              true,
	"temp":             true,
	"logs":             true,
	"log":              true,
	".npm":             true,
	"coverage":         true,
	".next":            true,
	".nuxt":            true,
	".cache":           true,
	".parcel-cache":    true,
}

// IgnoreExts contains file extensions to skip during search
var IgnoreExts = map[string]bool{
	".log":           true,
	".lock":          true,
	".min.js":        true,
	".min.css":       true,
	".map":           true,
	".DS_Store":      true,
	"Thumbs.db":      true,
	".gitignore":     true,
	".gitattributes": true,
	".eslintrc":      true,
	".prettierrc":    true,
}

// BinaryExts contains extensions of files to skip due to binary content
var BinaryExts = map[string]bool{
	".bin": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
}

// LoadConfig parses command-line flags and sets up configuration
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	flag.Int64Var(&cfg.MaxFileSize, "max-size", 1024*1024, "Maximum file size in bytes")
	flag.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "Timeout for operation")
	flag.IntVar(&cfg.Workers, "workers", 10, "Number of concurrent workers")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&cfg.Debug, "debug", true, "Enable debug mode")
	flag.IntVar(&cfg.MaxMatches, "max-matches", 15, "Maximum number of fuzzy matches to display")
	flag.IntVar(&cfg.SearchDepth, "depth", 5, "Maximum depth to search for fuzzy matches")
	flag.BoolVar(&cfg.AutoSelect, "auto", false, "Automatically select best match if score is good enough")
	flag.BoolVar(&cfg.SearchHidden, "hidden", false, "Include hidden files in search")
	flag.BoolVar(&cfg.NoIgnore, "no-ignore", false, "Don't skip common ignored directories")

	// Setup debug log file
	var err error
	cfg.LogFile, err = os.Create("fcopy_debug.log")
	if err != nil {
		return cfg, err
	}

	cfg.Logger = log.New(cfg.LogFile, "", log.LstdFlags)

	return cfg, nil
}
