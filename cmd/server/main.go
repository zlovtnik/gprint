package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/zlovtnik/gprint/internal/config"
	"github.com/zlovtnik/gprint/internal/handlers"
	"github.com/zlovtnik/gprint/internal/repository"
	"github.com/zlovtnik/gprint/internal/router"
	"github.com/zlovtnik/gprint/internal/service"
	"github.com/zlovtnik/gprint/pkg/auth"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg, logger := loadConfigAndLogger()

	db := setupDatabase(cfg, logger)

	repos := setupRepositories(db)

	services := setupServices(repos, cfg, logger)

	handlers := setupHandlers(services, db, cfg)

	r, err := setupRouter(cfg, logger, handlers)
	if err != nil {
		logger.Error("failed to setup router", "error", err)
		os.Exit(1)
	}

	server := setupServer(cfg, r)

	cancel, bgWg := startBackgroundJobs(services.printSvc, cfg, logger)

	serverErrCh := startServer(server, logger)

	exitCode := waitForShutdown(server, db, cancel, bgWg, serverErrCh, logger, cfg)

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func loadConfigAndLogger() (*config.Config, *slog.Logger) {
	// Load configuration first so we can use it for logger setup
	cfg := config.Load()

	// Parse log level from configuration
	logLevel, ok := parseLogLevel(cfg.LogLevel)
	if !ok {
		fmt.Fprintf(os.Stderr, "WARNING: unknown log level %q, defaulting to info\n", cfg.LogLevel)
	}

	// Initialize logger with configurable level
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("starting gprint service",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
	)

	return cfg, logger
}

func setupDatabase(cfg *config.Config, logger *slog.Logger) *sql.DB {
	// Connect to database
	db, err := config.NewOracleDB(cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	// Note: db.Close() is called explicitly during graceful shutdown
	logger.Info("connected to database")
	return db
}

// repositories holds all repository instances
type repositories struct {
	customerRepo           *repository.CustomerRepository
	serviceRepo            *repository.ServiceRepository
	contractRepo           *repository.ContractRepository
	historyRepo            *repository.HistoryRepository
	printJobRepo           *repository.PrintJobRepository
	contractGenerationRepo *repository.ContractGenerationRepository
}

// services holds all service instances
type services struct {
	customerSvc           *service.CustomerService
	serviceSvc            *service.ServiceService
	contractSvc           *service.ContractService
	printSvc              *service.PrintService
	contractGenerationSvc *service.ContractGenerationService
}

// handlerSet holds all handler instances
type handlerSet struct {
	customerHandler           *handlers.CustomerHandler
	serviceHandler            *handlers.ServiceHandler
	contractHandler           *handlers.ContractHandler
	contractGenerationHandler *handlers.ContractGenerationHandler
	printHandler              *handlers.PrintHandler
	healthHandler             *handlers.HealthHandler
	authHandler               *handlers.AuthHandler
}

func setupRepositories(db *sql.DB) repositories {
	// Initialize repositories
	customerRepo := repository.NewCustomerRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	contractRepo := repository.NewContractRepository(db)
	historyRepo := repository.NewHistoryRepository(db)
	printJobRepo := repository.NewPrintJobRepository(db)
	contractGenerationRepo := repository.NewContractGenerationRepository(db)

	return repositories{
		customerRepo:           customerRepo,
		serviceRepo:            serviceRepo,
		contractRepo:           contractRepo,
		historyRepo:            historyRepo,
		printJobRepo:           printJobRepo,
		contractGenerationRepo: contractGenerationRepo,
	}
}

func setupServices(repos repositories, cfg *config.Config, logger *slog.Logger) services {
	// Initialize services
	customerSvc := service.NewCustomerService(repos.customerRepo)
	serviceSvc := service.NewServiceService(repos.serviceRepo)
	contractSvc := service.NewContractService(repos.contractRepo, repos.historyRepo)
	printSvc, err := service.NewPrintService(repos.printJobRepo, repos.contractRepo, repos.historyRepo, cfg.Print.OutputPath, logger)
	if err != nil {
		logger.Error("failed to create print service", "error", err)
		os.Exit(1)
	}
	contractGenerationSvc := service.NewContractGenerationService(repos.contractGenerationRepo)

	return services{
		customerSvc:           customerSvc,
		serviceSvc:            serviceSvc,
		contractSvc:           contractSvc,
		printSvc:              printSvc,
		contractGenerationSvc: contractGenerationSvc,
	}
}

func setupHandlers(svcs services, db *sql.DB, cfg *config.Config) handlerSet {
	// Validate Keycloak configuration before creating client
	if cfg.Keycloak.BaseURL == "" {
		panic("KEYCLOAK_URL is required for authentication")
	}
	if cfg.Keycloak.Realm == "" {
		panic("KEYCLOAK_REALM is required for authentication")
	}
	if cfg.Keycloak.ClientID == "" {
		panic("KEYCLOAK_CLIENT_ID is required for authentication")
	}

	// Initialize Keycloak client
	keycloakClient := auth.NewKeycloakClient(auth.KeycloakConfig{
		BaseURL:      cfg.Keycloak.BaseURL,
		Realm:        cfg.Keycloak.Realm,
		ClientID:     cfg.Keycloak.ClientID,
		ClientSecret: cfg.Keycloak.ClientSecret,
	})

	// Initialize handlers
	customerHandler := handlers.NewCustomerHandler(svcs.customerSvc)
	serviceHandler := handlers.NewServiceHandler(svcs.serviceSvc)
	contractHandler := handlers.NewContractHandler(svcs.contractSvc)
	contractGenerationHandler := handlers.NewContractGenerationHandler(svcs.contractGenerationSvc)
	printHandler := handlers.NewPrintHandler(svcs.printSvc)
	healthHandler := handlers.NewHealthHandler(db)
	authHandler := handlers.NewAuthHandler(keycloakClient, cfg.JWT.Secret)

	return handlerSet{
		customerHandler:           customerHandler,
		serviceHandler:            serviceHandler,
		contractHandler:           contractHandler,
		contractGenerationHandler: contractGenerationHandler,
		printHandler:              printHandler,
		healthHandler:             healthHandler,
		authHandler:               authHandler,
	}
}

func setupRouter(cfg *config.Config, logger *slog.Logger, h handlerSet) (*router.Router, error) {
	// Initialize router
	r, err := router.NewRouter(
		cfg.JWT.Secret,
		logger,
		router.Handlers{
			Customer:           h.customerHandler,
			Service:            h.serviceHandler,
			Contract:           h.contractHandler,
			ContractGeneration: h.contractGenerationHandler,
			Print:              h.printHandler,
			Health:             h.healthHandler,
			Auth:               h.authHandler,
		},
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func setupServer(cfg *config.Config, r *router.Router) *http.Server {
	// Create HTTP server
	server := &http.Server{
		Addr:           cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:        r.Setup(),
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}
	return server
}

func startBackgroundJobs(printSvc *service.PrintService, cfg *config.Config, logger *slog.Logger) (context.CancelFunc, *sync.WaitGroup) {
	// Start background print job processor
	ctx, cancel := context.WithCancel(context.Background())

	// WaitGroup to track the background goroutine for graceful shutdown
	var wg sync.WaitGroup

	// Mutex to prevent overlapping ProcessPendingJobs executions
	var jobMu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done() // Ensure Done runs after any in-flight ProcessPendingJobs completes

		// Process pending jobs immediately on startup
		jobMu.Lock()
		if err := printSvc.ProcessPendingJobs(ctx); err != nil {
			logger.Error("failed to process pending print jobs on startup", "error", err)
		}
		jobMu.Unlock()

		ticker := time.NewTicker(cfg.Print.JobInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Skip this tick if previous job is still running
				if !jobMu.TryLock() {
					logger.Debug("skipping print job tick, previous job still running")
					continue
				}
				if err := printSvc.ProcessPendingJobs(ctx); err != nil {
					logger.Error("failed to process pending print jobs", "error", err)
				}
				jobMu.Unlock()
			}
		}
	}()

	return cancel, &wg
}

func startServer(server *http.Server, logger *slog.Logger) chan error {
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

	return serverErrCh
}

func waitForShutdown(server *http.Server, db *sql.DB, cancel context.CancelFunc, bgWg *sync.WaitGroup, serverErrCh chan error, logger *slog.Logger, cfg *config.Config) int {
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

	// Cancel background jobs and wait for them to finish before closing DB
	cancel()
	logger.Debug("waiting for background jobs to complete...")
	bgWg.Wait()
	logger.Debug("background jobs completed")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
		exitCode = 1
	}

	// Explicitly close database after background jobs have finished using it
	if err := db.Close(); err != nil {
		logger.Error("database close error", "error", err)
	}

	logger.Info("server stopped")

	return exitCode
}

// parseLogLevel parses a log level string into slog.Level
// Returns the level and true if recognized, or LevelInfo and false if unknown
func parseLogLevel(level string) (slog.Level, bool) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}
