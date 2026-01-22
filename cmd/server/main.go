package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zlovtnik/gprint/internal/config"
	"github.com/zlovtnik/gprint/internal/handlers"
	"github.com/zlovtnik/gprint/internal/repository"
	"github.com/zlovtnik/gprint/internal/router"
	"github.com/zlovtnik/gprint/internal/service"
)

func main() {
	// Load configuration first so we can use it for logger setup
	cfg := config.Load()

	// Parse log level from configuration
	logLevel := parseLogLevel(cfg.LogLevel)

	// Initialize logger with configurable level
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("starting gprint service",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
	)

	// Connect to database
	db, err := config.NewOracleDB(cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	// Note: db.Close() is called explicitly during graceful shutdown
	logger.Info("connected to database")

	// Initialize repositories
	customerRepo := repository.NewCustomerRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	contractRepo := repository.NewContractRepository(db)
	historyRepo := repository.NewHistoryRepository(db)
	printJobRepo := repository.NewPrintJobRepository(db)

	// Initialize services
	customerSvc := service.NewCustomerService(customerRepo)
	serviceSvc := service.NewServiceService(serviceRepo)
	contractSvc := service.NewContractService(contractRepo, historyRepo)
	printSvc, err := service.NewPrintService(printJobRepo, contractRepo, historyRepo, cfg.Print.OutputPath, logger)
	if err != nil {
		logger.Error("failed to create print service", "error", err)
		os.Exit(1)
	}

	// Initialize handlers
	customerHandler := handlers.NewCustomerHandler(customerSvc)
	serviceHandler := handlers.NewServiceHandler(serviceSvc)
	contractHandler := handlers.NewContractHandler(contractSvc)
	printHandler := handlers.NewPrintHandler(printSvc)
	healthHandler := handlers.NewHealthHandler(db)

	// Initialize router
	r := router.NewRouter(
		cfg.JWT.Secret,
		logger,
		customerHandler,
		serviceHandler,
		contractHandler,
		printHandler,
		healthHandler,
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      r.Setup(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start background print job processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// Process pending jobs immediately on startup
		if err := printSvc.ProcessPendingJobs(ctx); err != nil {
			logger.Error("failed to process pending print jobs on startup", "error", err)
		}

		ticker := time.NewTicker(cfg.Print.JobInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := printSvc.ProcessPendingJobs(ctx); err != nil {
					logger.Error("failed to process pending print jobs", "error", err)
				}
			}
		}
	}()

	// Error channel for server listen errors
	serverErrCh := make(chan error, 1)

	// Start server in goroutine
	go func() {
		logger.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			serverErrCh <- err
		}
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	exitCode := 0
	select {
	case <-quit:
		logger.Info("received shutdown signal")
	case err := <-serverErrCh:
		logger.Error("server listen failed", "error", err)
		exitCode = 1
	}

	logger.Info("shutting down server...")

	// Cancel background jobs
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
		exitCode = 1
	}

	// Explicitly close database before exit
	if err := db.Close(); err != nil {
		logger.Error("database close error", "error", err)
	}

	logger.Info("server stopped")

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// parseLogLevel parses a log level string into slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
