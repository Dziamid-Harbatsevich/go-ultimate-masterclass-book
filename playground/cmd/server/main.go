package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type AppEnv struct {
	Log *slog.Logger
}

// logsDir is where NDJSON log files are persisted (relative to the module root).
const logsDir = "cmd/server/logs"

// levelSplitHandler routes each record to a file based on its level:
// DEBUG/INFO -> access.log, WARN/ERROR -> errors.log.
type levelSplitHandler struct {
	access slog.Handler
	errors slog.Handler
}

func (h *levelSplitHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.access.Enabled(ctx, level) || h.errors.Enabled(ctx, level)
}

func (h *levelSplitHandler) Handle(ctx context.Context, r slog.Record) error {
	var err error
	if r.Level >= slog.LevelWarn {
		err = h.errors.Handle(ctx, r)
	} else {
		err = h.access.Handle(ctx, r)
	}
	if err != nil {
		// Never break the caller because of file IO problems.
		fmt.Fprintf(os.Stderr, "log write failed: %v\n", err)
	}
	return nil
}

func (h *levelSplitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelSplitHandler{access: h.access.WithAttrs(attrs), errors: h.errors.WithAttrs(attrs)}
}

func (h *levelSplitHandler) WithGroup(name string) slog.Handler {
	return &levelSplitHandler{access: h.access.WithGroup(name), errors: h.errors.WithGroup(name)}
}

// teeHandler writes every record to multiple handlers (serialized with a mutex,
// since os.File writes are not atomic).
type teeHandler struct {
	mu       sync.Mutex
	handlers []slog.Handler
}

func (h *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, inner := range h.handlers {
		if inner.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, inner := range h.handlers {
		if !inner.Enabled(ctx, r.Level) {
			continue
		}
		if err := inner.Handle(ctx, r.Clone()); err != nil {
			fmt.Fprintf(os.Stderr, "log write failed: %v\n", err)
		}
	}
	return nil
}

func (h *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clones := make([]slog.Handler, len(h.handlers))
	for i, inner := range h.handlers {
		clones[i] = inner.WithAttrs(attrs)
	}
	return &teeHandler{handlers: clones}
}

func (h *teeHandler) WithGroup(name string) slog.Handler {
	clones := make([]slog.Handler, len(h.handlers))
	for i, inner := range h.handlers {
		clones[i] = inner.WithGroup(name)
	}
	return &teeHandler{handlers: clones}
}

// openLogFiles creates the logs folder if needed and opens access.log / errors.log for appending.
func openLogFiles() (accessFile, errorsFile *os.File, err error) {
	if err = os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, nil, err
	}
	accessFile, err = os.OpenFile(filepath.Join(logsDir, "access.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	errorsFile, err = os.OpenFile(filepath.Join(logsDir, "errors.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		_ = accessFile.Close()
		return nil, nil, err
	}
	return accessFile, errorsFile, nil
}

// setupLogger builds a console+file logger and installs it as the global default.
// The returned func closes the log files and must be deferred.
func setupLogger() (*slog.Logger, func(), error) {
	accessFile, errorsFile, err := openLogFiles()
	if err != nil {
		return nil, nil, err
	}

	fileOptions := &slog.HandlerOptions{Level: slog.LevelDebug}
	split := &levelSplitHandler{
		access: slog.NewJSONHandler(accessFile, fileOptions),
		errors: slog.NewJSONHandler(errorsFile, fileOptions),
	}

	logger := slog.New(&teeHandler{handlers: []slog.Handler{
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		split,
	}})
	slog.SetDefault(logger)

	var closeOnce sync.Once
	cleanup := func() {
		closeOnce.Do(func() {
			_ = accessFile.Close()
			_ = errorsFile.Close()
		})
	}
	return logger, cleanup, nil
}

func main() {
	// 1. Initialize the system's global JSON structured logger (console + files)
	logger, closeLogs, err := setupLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize file logging: %v\n", err)
		os.Exit(1)
	}
	defer closeLogs()

	logger.Info("Bootstrapping enterprise microservice architecture stack...")

	env := &AppEnv{Log: logger}
	mux := http.NewServeMux()

	// 2. Define business routes and a panic-testing validation route
	mux.HandleFunc("GET /api/v1/health", env.HealthCheckHandler)
	mux.HandleFunc("GET /api/v1/panic-test", env.SimulatePanicHandler)

	// 3. Chain middlewares in reverse execution order
	middlewarePipeline := RecoveryMiddleware(logger, mux)
	middlewarePipeline = StructuredLoggingMiddleware(logger, middlewarePipeline)
	middlewarePipeline = TraceMiddleware(middlewarePipeline)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      middlewarePipeline,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// 4. Set up an operating system kernel termination listener channel
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		logger.Info("Production REST engine listening securely", slog.String("port", "8080"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Critical network socket crash captured", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Execution halts right here until a shutdown signal is intercepted
	<-shutdownChan
	logger.Warn("🛑 Kernel termination signal intercepted. Initializing connection draining sequence...")

	// Allow current connections a strict 10-second grace window to finish active transactions
	drainContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(drainContext); err != nil {
		logger.Error("Forceful server termination required", slog.Any("error", err))
		_ = server.Close()
	}

	logger.Info("✨ Microservice connection pools drained. System exit clean.")
}

func (env *AppEnv) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "operational",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (env *AppEnv) SimulatePanicHandler(w http.ResponseWriter, r *http.Request) {
	// Intentionally trigger a runtime nil-pointer dereference to test middleware recovery
	var unsafePointer *string
	fmt.Println(*unsafePointer)
}
