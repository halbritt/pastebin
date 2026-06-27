package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"pastebin/internal/paste"
	"pastebin/internal/server"
	sqlitestore "pastebin/internal/storage/sqlite"
)

const defaultListen = "127.0.0.1:8080"

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Printf("pastebind: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}

	store, err := sqlitestore.Open(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	handler, err := server.New(server.Config{
		Store:      store,
		BaseURL:    cfg.BaseURL,
		MaxBytes:   cfg.MaxBytes,
		DefaultTTL: cfg.DefaultTTL,
		MaxTTL:     cfg.MaxTTL,
	})
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", cfg.Listen)
		errCh <- httpServer.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}

type config struct {
	BaseURL    string
	Listen     string
	DBPath     string
	MaxBytes   int64
	DefaultTTL time.Duration
	MaxTTL     time.Duration
}

func parseConfig(args []string) (config, error) {
	flags := flag.NewFlagSet("pastebind", flag.ContinueOnError)
	envMaxBytes, err := getenvInt64("PASTEBIN_MAX_BYTES", paste.DefaultMaxBytes)
	if err != nil {
		return config{}, err
	}
	envDefaultTTL, err := getenvDuration("PASTEBIN_DEFAULT_TTL", paste.DefaultTTL)
	if err != nil {
		return config{}, err
	}
	envMaxTTL, err := getenvDuration("PASTEBIN_MAX_TTL", paste.MaxTTL)
	if err != nil {
		return config{}, err
	}

	baseURL := flags.String("base-url", getenv("PASTEBIN_BASE_URL", "http://"+defaultListen), "base URL used in paste receipts")
	listen := flags.String("listen", getenv("PASTEBIN_LISTEN", defaultListen), "HTTP listen address")
	dbPath := flags.String("db", getenv("PASTEBIN_DB", "pastebin.db"), "SQLite database path")
	maxBytes := flags.Int64("max-bytes", envMaxBytes, "maximum paste size in bytes")
	defaultTTL := flags.Duration("default-ttl", envDefaultTTL, "default paste TTL")
	maxTTL := flags.Duration("max-ttl", envMaxTTL, "maximum paste TTL")
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if strings.TrimSpace(*listen) == "" {
		return config{}, errors.New("listen address is required")
	}
	if strings.TrimSpace(*dbPath) == "" {
		return config{}, errors.New("database path is required")
	}
	if *maxBytes <= 0 {
		return config{}, errors.New("max bytes must be positive")
	}
	if *maxTTL <= 0 {
		return config{}, errors.New("max ttl must be positive")
	}
	if _, err := paste.ValidateTTL(*defaultTTL, *defaultTTL, *maxTTL); err != nil {
		return config{}, fmt.Errorf("default ttl: %w", err)
	}
	return config{
		BaseURL:    strings.TrimRight(strings.TrimSpace(*baseURL), "/"),
		Listen:     strings.TrimSpace(*listen),
		DBPath:     strings.TrimSpace(*dbPath),
		MaxBytes:   *maxBytes,
		DefaultTTL: *defaultTTL,
		MaxTTL:     *maxTTL,
	}, nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getenvInt64(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
