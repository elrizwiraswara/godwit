package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	api "github.com/elrizwiraswara/godwit/api"
	config "github.com/elrizwiraswara/godwit/config"
	embedding "github.com/elrizwiraswara/godwit/embedding"
)

func main() {
	// Usage: go run ./server
	//
	// Environment variables (flags override the first two):
	// GODWIT_ADDR          listen address (default 127.0.0.1:8080)
	// GODWIT_MODEL_DIR     embedding model directory
	// GODWIT_API_KEYS      comma-separated API keys; required unless listening on localhost
	// GODWIT_CORS_ORIGINS  comma-separated browser origins allowed to call the API
	// GODWIT_MAX_BODY_MB   request size limit in MB (default 32)

	addr := flag.String("addr", envOr("GODWIT_ADDR", "127.0.0.1:8080"), "listen address; port 0 picks a free port")
	modelDir := flag.String("model", os.Getenv("GODWIT_MODEL_DIR"), "embedding model directory")
	healthcheck := flag.Bool("healthcheck", false, "check that the server at -addr is healthy, then exit")
	exitOnStdinClose := flag.Bool("exit-on-stdin-close", false, "stop when stdin closes (used by the desktop app)")
	flag.Parse()

	if *healthcheck {
		os.Exit(checkHealth(*addr))
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Error listening on %s: %v", *addr, err)
	}

	apiKeys := splitList(os.Getenv("GODWIT_API_KEYS"))
	if len(apiKeys) == 0 {
		if !isLoopback(listener.Addr()) {
			log.Fatalf("GODWIT_API_KEYS must be set when listening on %s", listener.Addr())
		}
		log.Print("WARNING: GODWIT_API_KEYS is not set; accepting requests without an API key on localhost")
	}

	cfg := api.Config{APIKeys: apiKeys, CORSOrigins: splitList(os.Getenv("GODWIT_CORS_ORIGINS"))}
	if mb, err := strconv.Atoi(os.Getenv("GODWIT_MAX_BODY_MB")); err == nil && mb > 0 {
		cfg.MaxBodyBytes = int64(mb) << 20
	}
	if model, err := embedding.Load(resolveModelDir(*modelDir)); err != nil {
		log.Printf("Embedding model not loaded (%v), continuing with rules only", err)
	} else {
		cfg.Embedder = model
	}

	srv := &http.Server{
		Handler:           api.New(cfg),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *exitOnStdinClose {
		// The desktop app keeps stdin open; if the app exits or crashes, stop too
		go func() {
			io.Copy(io.Discard, os.Stdin)
			stop()
		}()
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	// Machine-readable line the desktop app waits for to learn the port
	fmt.Printf("LISTENING %s\n", listener.Addr())
	log.Printf("Serving on http://%s", listener.Addr())

	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server error: %v", err)
	}
}

// resolveModelDir prefers an explicit directory, then the model next to the
// executable (installed app, Docker image), then the repository layout.
func resolveModelDir(dir string) string {
	if dir != "" {
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), config.EMBEDDING_MODEL_DIR)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return config.EMBEDDING_MODEL_DIR
}

func checkHealth(addr string) int {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 1
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return 1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func isLoopback(addr net.Addr) bool {
	tcp, ok := addr.(*net.TCPAddr)
	return ok && tcp.IP.IsLoopback()
}

func splitList(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
