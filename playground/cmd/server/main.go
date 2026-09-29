package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type AppEnv struct {
	Log *slog.Logger
}

func main() {
	// 1. Initialize the system's global JSON structured logger
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

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
