# fcopy

**fcopy** is a fast and flexible file processing tool written in Go. It is designed for efficiently finding, processing, and extracting file content from both files and directories. By combining fuzzy searching, concurrency, and customizable configurations, **fcopy** is especially useful when preparing large codebases or document collections for further processing, such as feeding into large language models (LLMs) for analysis or summarization.

## Features

- **Fuzzy Path Matching:**  
  Uses a combination of substring matching and the Levenshtein distance algorithm to locate files and directories by approximate names. This is invaluable when dealing with large codebases where spelling variations or imprecise input might otherwise hinder file discovery.

- **Recursive Directory Processing:**  
  Efficiently processes directories by walking them recursively while respecting configurable limits, such as maximum search depth and file size.

- **Concurrency:**  
  Utilizes a worker pool mechanism to process files in parallel, dramatically improving throughput in large-scale file systems.

- **Customizable Configuration:**  
  Easily configurable via command-line flags. Options include maximum file size, operation timeout, number of workers, verbosity, and advanced fuzzy matching parameters.

- **Ignore Rules:**  
  Automatically skips common directories (like `.git`, `node_modules`, etc.) and file types (such as logs, binaries, or minimized assets) to ensure that processing focuses only on relevant content. Users can opt-in to include hidden files or override the ignore functionality entirely.

- **Debug Logging:**  
  Generates detailed debug logs to a file (`fcopy_debug.log`), making it easier to diagnose issues during file scanning or processing.

## How It Helps with LLMs

Large Language Models rely on high-quality, curated input data. **fcopy** simplifies the preparation process by:
  
- **Extracting Relevant Content:**  
  Quickly locating and reading file contents across extensive directory structures. This is especially useful for generating context data, documentation summaries, or code snippets from legacy projects.
  
- **Reducing Noise:**  
  By ignoring non-essential directories and binary files, **fcopy** ensures only the most relevant information is passed on. This clean data can enhance the accuracy and efficiency of LLM-based processing tasks.
  
- **Customizable Preprocessing:**  
  Developers can fine-tune the search parameters, file size limits, and processing behavior, providing the flexibility needed to craft high-quality datasets for training, fine-tuning, or query-based applications with LLMs.

## Underlying Algorithms and Design

- **Fuzzy Matching:**  
  The project uses a combination of substring checks and the Levenshtein distance algorithm to determine the similarity between file/directory names and user queries. The Levenshtein distance is calculated in the `utils` package to assign a match score, with lower scores indicating more similar strings.

- **Directory Traversal:**  
  Recursion via `filepath.WalkDir` allows for efficient exploration of complex directory structures. The tool also enforces a configurable search depth, minimizing unnecessary traversal in large directory trees.

- **Concurrent File Processing:**  
  A worker pool implementation processes files concurrently, improving performance when dealing with I/O-bound tasks. This design helps the tool scale well even for large projects.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.16 or higher

### Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/not-ani/fcopy.git
   cd fcopy
   ```

2. **Install the cli:**

   ```bash
   cd cmd/fcopy
   go build -o fcopy
   go install
   ```

### Usage

After building, you can run **fcopy** from the command line. Here are some example flags:

```bash
./fcopy --max-size=1048576 --timeout=30s --workers=10 --verbose --max-matches=15 --depth=5 --auto --hidden --no-ignore
```

- `--max-size`: Maximum file size in bytes.
- `--timeout`: Operation timeout duration.
- `--workers`: Number of concurrent processing workers.
- `--verbose`: Enable verbose output.
- `--max-matches`: Maximum number of fuzzy matches to display.
- `--depth`: Maximum search depth for fuzzy matching.
- `--auto`: Automatically select the best match if it meets quality criteria.
- `--hidden`: Include hidden files in the search.
- `--no-ignore`: Do not skip common ignored directories.

## Contributing

Contributions are always welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to get started.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact

For any questions or suggestions, please open an issue on GitHub or reach out directly.
