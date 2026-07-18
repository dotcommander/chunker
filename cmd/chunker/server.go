package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dotcommander/chunker/internal/handler"
	"github.com/dotcommander/chunker/internal/service"
)

func runServerMode(portStr string) {
	chunkService := newChunkService()
	chunkHandler := handler.NewChunkHandler(chunkService)

	router := setupRouter(chunkHandler)
	srv := createServer(portStr, router)
	errCh := startServer(srv, portStr)
	if err := waitForShutdown(srv, errCh); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func newChunkService() *service.ChunkService {
	return service.NewChunkService()
}

func resolvePort(cliPort string) string {
	if envPort := os.Getenv("PORT"); envPort != "" {
		return envPort
	}
	return cliPort
}

func setupRouter(chunkHandler *handler.ChunkHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Post("/chunk", chunkHandler.HandleChunk)
	r.Get("/health", chunkHandler.HandleHealth)

	return r
}

func createServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second, // slowloris defense: cap header read
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      65 * time.Second, // > middleware.Timeout(60s) so the handler budget fires first
		IdleTimeout:       120 * time.Second,
	}
}

// startServer launches the HTTP listener in a goroutine and returns a buffered
// channel that receives ListenAndServe's error (or nil if the listener exited
// via http.ErrServerClosed). waitForShutdown selects across this channel so
// bind failures surface immediately instead of being swallowed.
func startServer(srv *http.Server, port string) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Starting server on port %s", port)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	return errCh
}

func waitForShutdown(srv *http.Server, errCh <-chan error) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		// Listener exited before any shutdown signal. Bind failure or
		// other startup error — surface it immediately.
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-quit:
	}

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
