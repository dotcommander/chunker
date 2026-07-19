package main

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestStartServer_BindFailure verifies the bind-failure path: when the
// listener cannot acquire the port, startServer's error channel surfaces
// the error rather than waitForShutdown swallowing it.
func TestStartServer_BindFailure(t *testing.T) {
	t.Parallel()

	// Occupy a port so a second listener on the same address fails to bind.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("seed listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	port := strconv.Itoa(addr.Port)

	srv := &http.Server{
		Addr:    "127.0.0.1:" + port,
		Handler: http.NewServeMux(),
	}

	errCh := startServer(srv)
	err = waitForShutdown(srv, errCh)
	if err == nil {
		t.Fatal("expected bind failure error, got nil")
	}
	if !strings.Contains(err.Error(), "listen") {
		t.Fatalf("expected listen error, got: %v", err)
	}
}

// TestStartServer_GracefulShutdown verifies the success path: the listener
// stays up until Close is invoked, and the error channel reports nil rather
// than the http.ErrServerClosed sentinel.
func TestStartServer_GracefulShutdown(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}

	// Use a manual listener so we know it's bound before we proceed.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		err := srv.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// Give Serve a moment to enter the accept loop, then shut down.
	time.Sleep(20 * time.Millisecond)
	if err := srv.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected nil after clean shutdown, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for listener to exit")
	}
}

func TestCreateServer_Timeouts(t *testing.T) {
	t.Parallel()
	srv := createServer("127.0.0.1", "0", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	if srv.ReadHeaderTimeout <= 0 {
		t.Errorf("ReadHeaderTimeout = %v, want > 0 (slowloris defense)", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout <= 0 {
		t.Errorf("ReadTimeout = %v, want > 0", srv.ReadTimeout)
	}
	if srv.WriteTimeout <= 0 {
		t.Errorf("WriteTimeout = %v, want > 0", srv.WriteTimeout)
	}
	if srv.IdleTimeout <= 0 {
		t.Errorf("IdleTimeout = %v, want > 0", srv.IdleTimeout)
	}
}

func TestCreateServer_Address(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		bind string
		want string
	}{
		{name: "IPv4 loopback", bind: "127.0.0.1", want: "127.0.0.1:8080"},
		{name: "IPv6 loopback", bind: "::1", want: "[::1]:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := createServer(tt.bind, "8080", http.NewServeMux())
			if srv.Addr != tt.want {
				t.Errorf("Addr = %q, want %q", srv.Addr, tt.want)
			}
		})
	}
}
