package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"chunker/internal/domain"
	"chunker/internal/handler"
	"chunker/internal/service"
	"chunker/pkg/chunking"
)

var (
	serverMode    = flag.Bool("server", false, "Run in server mode")
	port          = flag.String("port", "8080", "Server port (server mode only)")
	chunkSize     = flag.Int("size", 4000, "Chunk size (characters or tokens)")
	strategy      = flag.String("strategy", "", "Chunking strategy (defaults to smart_boundary)")
	overlap       = flag.Int("overlap", 200, "Overlap between chunks")
	tokenEncoding = flag.String("encoding", "", "Token encoding (for token_based strategy)")
	outputFormat  = flag.String("format", "json", "Output format: json, jsonl")
	pretty        = flag.Bool("pretty", false, "Pretty print JSON output")
	help          = flag.Bool("h", false, "Show help")
)

func main() {
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Check if stdin has data (for pipe mode)
	stat, _ := os.Stdin.Stat()
	isPiped := (stat.Mode() & os.ModeCharDevice) == 0

	// If no flags and stdin is piped, run in CLI mode
	if !*serverMode && isPiped {
		runCLIMode()
	} else if *serverMode || !isPiped {
		runServerMode()
	} else {
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `Chunker - Smart text chunking service

Usage:
  # CLI mode (pipe text through stdin):
  cat document.txt | chunker
  echo "text" | chunker -size 1000 -strategy token_based -encoding cl100k_base
  
  # Server mode:
  chunker -server
  chunker -server -port 3000

CLI Options:
  -size       Chunk size in characters or tokens (default: 4000)
  -strategy   Chunking strategy: smart_boundary, word_boundary, sentence_boundary,
              hard_cut, paragraph_aware, token_based (default: smart_boundary)
  -overlap    Overlap between chunks (default: 200)
  -encoding   Token encoding for token_based strategy: cl100k_base, o200k_base,
              p50k_base, r50k_base (default: cl100k_base)
  -format     Output format: json, jsonl (default: json)
  -pretty     Pretty print JSON output

Server Options:
  -server     Run in server mode
  -port       Server port (default: 8080)

Examples:
  # Basic chunking with smart defaults
  cat article.txt | chunker
  
  # Token-based chunking for LLM processing
  cat code.py | chunker -size 2000 -strategy token_based -encoding cl100k_base
  
  # Output as JSON Lines for streaming
  cat book.txt | chunker -format jsonl
  
  # Custom overlap for context preservation
  echo "Long text..." | chunker -size 1000 -overlap 100
`)
}

func runCLIMode() {
	// Read all input from stdin
	reader := bufio.NewReader(os.Stdin)
	var builder strings.Builder
	
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				builder.WriteString(text)
				break
			}
			log.Fatalf("Error reading stdin: %v", err)
		}
		builder.WriteString(text)
	}

	inputText := builder.String()
	if inputText == "" {
		log.Fatal("No input text provided")
	}

	// Initialize chunking service
	factory := chunking.NewFactory()
	chunkService := service.NewChunkService(factory)

	// Prepare request
	req := domain.ChunkRequest{
		Text:      inputText,
		ChunkSize: *chunkSize,
		Overlap:   *overlap,
	}

	if *strategy != "" {
		req.Strategy = domain.Strategy(*strategy)
	}

	if *tokenEncoding != "" {
		req.TokenEncoding = domain.TokenEncoding(*tokenEncoding)
	}

	// Process chunks
	ctx := context.Background()
	resp, err := chunkService.ProcessChunkRequest(ctx, req)
	if err != nil {
		log.Fatalf("Error chunking text: %v", err)
	}

	// Output results
	switch *outputFormat {
	case "jsonl":
		outputJSONL(resp)
	default:
		outputJSON(resp)
	}
}

func outputJSON(resp *domain.ChunkResponse) {
	encoder := json.NewEncoder(os.Stdout)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	
	if err := encoder.Encode(resp); err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
	}
}

func outputJSONL(resp *domain.ChunkResponse) {
	encoder := json.NewEncoder(os.Stdout)
	
	// Output metadata first
	metadata := map[string]interface{}{
		"type":     "metadata",
		"metadata": resp.Metadata,
	}
	if err := encoder.Encode(metadata); err != nil {
		log.Fatalf("Error encoding metadata: %v", err)
	}
	
	// Output each chunk as a separate line
	for _, chunk := range resp.Chunks {
		chunkLine := map[string]interface{}{
			"type":  "chunk",
			"chunk": chunk,
		}
		if err := encoder.Encode(chunkLine); err != nil {
			log.Fatalf("Error encoding chunk: %v", err)
		}
	}
}

func runServerMode() {
	portStr := *port
	if envPort := os.Getenv("PORT"); envPort != "" {
		portStr = envPort
	}

	// Initialize dependencies
	factory := chunking.NewFactory()
	chunkService := service.NewChunkService(factory)
	chunkHandler := handler.NewChunkHandler(chunkService)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Post("/chunk", chunkHandler.HandleChunk)
	r.Get("/health", chunkHandler.HandleHealth)

	// Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", portStr),
		Handler: r,
	}

	// Start server
	go func() {
		log.Printf("Starting server on port %s", portStr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}