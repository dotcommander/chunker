package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dotcommander/chunker/internal/domain"
)

// Package-level flag pointers. Numeric defaults come from LoadConfig() —
// registerRootFlags binds them with config-driven defaults before flag.Parse.
// The pointers themselves are stable so subcommand FlagSets can re-bind to
// the same variables (see newChunkFlagSet).
var (
	chunkSize     = new(int)
	strategy      = new(string)
	overlap       = new(int)
	tokenEncoding = new(string)
	outputFormat  = new(string)
	pretty        = new(bool)
	help          = new(bool)

	// loadedConfig is populated once in main() before flag registration so
	// every subcommand sees the same defaults without re-reading the file.
	loadedConfig Config
)

func registerRootFlags(cfg Config) {
	flag.IntVar(chunkSize, "size", cfg.ChunkSize, "Chunk size (characters or tokens)")
	flag.StringVar(strategy, "strategy", "", "Chunking strategy (defaults to smart_boundary)")
	flag.IntVar(overlap, "overlap", cfg.Overlap, "Overlap between chunks")
	flag.StringVar(tokenEncoding, "encoding", "", "Token encoding (for token_based strategy)")
	flag.StringVar(outputFormat, "format", "json", "Output format: json, jsonl")
	flag.BoolVar(pretty, "pretty", false, "Pretty print JSON output")
	flag.BoolVar(help, "h", false, "Show help")
}

func main() {
	loadedConfig = LoadConfig()
	registerRootFlags(loadedConfig)

	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServeCommand(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "inspect" {
		runInspectCommand(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "files" {
		runFilesCommand(os.Args[2:])
		return
	}

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Check if stdin has data (for pipe mode)
	isPiped := isStdinPiped()

	// Route to appropriate mode
	if isPiped {
		runCLIMode()
	} else {
		printHelp()
		os.Exit(0)
	}
}

func runServeCommand(args []string) {
	serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
	servePort := serveFlags.String("port", loadedConfig.ServerPort, "Server port")
	serveHelp := serveFlags.Bool("h", false, "Show help")
	serveFlags.BoolVar(serveHelp, "help", false, "Show help")

	if err := serveFlags.Parse(args); err != nil {
		log.Fatal(err)
	}

	if *serveHelp {
		printServeHelp()
		os.Exit(0)
	}

	runServerMode(resolvePort(*servePort))
}

func isStdinPiped() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func runCLIMode() {
	chunkService := newChunkService()
	runner := NewCLIRunner(chunkService, os.Stdin, os.Stdout, os.Stderr)

	// Prepare flags
	flags := CLIFlags{
		ChunkSize:     *chunkSize,
		Overlap:       *overlap,
		Strategy:      *strategy,
		TokenEncoding: *tokenEncoding,
		OutputFormat:  *outputFormat,
		Pretty:        *pretty,
	}

	// Run CLI
	if err := runner.Run(context.Background(), flags); err != nil {
		log.Fatal(err)
	}
}

func newChunkFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.IntVar(chunkSize, "size", loadedConfig.ChunkSize, "Chunk size (characters or tokens)")
	fs.StringVar(strategy, "strategy", "", "Chunking strategy (defaults to smart_boundary)")
	fs.IntVar(overlap, "overlap", loadedConfig.Overlap, "Overlap between chunks")
	fs.StringVar(tokenEncoding, "encoding", "", "Token encoding (for token_based strategy)")
	fs.StringVar(outputFormat, "format", "json", "Output format: json, jsonl")
	fs.BoolVar(pretty, "pretty", false, "Pretty print JSON output")
	return fs
}

func chunkRequestFromFlags(text string) domain.ChunkRequest {
	return domain.ChunkRequest{
		Text:          text,
		ChunkSize:     *chunkSize,
		Overlap:       *overlap,
		Strategy:      domain.Strategy(*strategy),
		TokenEncoding: domain.TokenEncoding(*tokenEncoding),
	}
}

const helpText = `Chunker - Smart text chunking service

Usage:
	# Show help:
	chunker

  # CLI mode (pipe text through stdin):
  cat document.txt | chunker
  echo "text" | chunker -size 1000 -strategy token_based -encoding cl100k_base

  # Server mode:
	chunker serve
	chunker serve -port 3000

  # Inspect mode (stdin required):
  cat document.txt | chunker inspect

  # Batch mode:
  chunker files "docs/*.md" -out-dir chunks

CLI Options:
  -size       Chunk size in characters or tokens (default: 4000)
  -strategy   Chunking strategy: smart_boundary, word_boundary, sentence_boundary,
	              hard_cut, paragraph_aware, markdown_aware, token_based (default: smart_boundary)
  -overlap    Overlap between chunks (default: 200)
  -encoding   Token encoding for token_based strategy: cl100k_base, o200k_base,
              p50k_base, r50k_base (default: cl100k_base)
  -format     Output format: json, jsonl (default: json)
  -pretty     Pretty print JSON output

Server Options:
  serve subcommand:
  -port       Server port (default: 8080)

Inspect Options:
  inspect subcommand:
  -sample     Number of chunk previews to print (default: 2)
  -inspect-format  Inspect output format: ndjson, human (default: ndjson)

Files Options:
  files subcommand:
  -out-dir    Output directory (default: chunks)

Examples:
  # Basic chunking with smart defaults
  cat article.txt | chunker

  # Token-based chunking for LLM processing
  cat code.py | chunker -size 2000 -strategy token_based -encoding cl100k_base

  # Output as JSON Lines for streaming
  cat book.txt | chunker -format jsonl

  # Custom overlap for context preservation
  echo "Long text..." | chunker -size 1000 -overlap 100
`

func printHelp() {
	fmt.Fprint(os.Stderr, helpText)
}

func printServeHelp() {
	fmt.Fprint(os.Stderr, "Usage: chunker serve [-port 8080]\n")
}
