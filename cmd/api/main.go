package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jasimvs/sample-go-svc/config"
	"github.com/jasimvs/sample-go-svc/internal/detection"
	"github.com/jasimvs/sample-go-svc/internal/model"
	"github.com/jasimvs/sample-go-svc/internal/transaction"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cfgPath := "./config"
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	db, err := newSQLiteConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Error establishing database connection: %v", err)
	}
	defer func() {
		log.Println("Closing database connection pool...")
		if err := db.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()

	txRepo := transaction.NewSQLiteRepository(db)
	ctx := context.Background()
	if err := txRepo.Migrate(ctx); err != nil {
		log.Fatalf("Database migration failed: %v", err) //nolint:gocritic,exitAfterDefer
	}

	// --- Echo Instance & Middleware ---
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("1M")) // Good practice for POST

	// --- Handlers ---
	transactionChannel := make(chan model.Transaction)

	txService := transaction.NewService(txRepo, transactionChannel)
	txHandler := transaction.NewHandler(txService)

	// --- Routes ---
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to the Transaction API!")
	})
	apiGroup := e.Group("/api/v1")
	apiGroup.POST("/transaction", txHandler.CreateTransaction)

	detection.NewManager(transactionChannel, detection.NewHighVolumeRule()).RunInBackground()

	startServer(cfg, e)
}

func startServer(cfg config.Config, e *echo.Echo) {
	serverAddress := fmt.Sprintf(":%s", cfg.Server.Port)
	go func() {
		log.Printf("Starting server on %s", serverAddress)
		if err := e.Start(serverAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal("Shutting down the server: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Received shutdown signal. Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server gracefully stopped.")
}

func newSQLiteConnection(cfg config.Database) (*sql.DB, error) {
	dbDir := filepath.Dir(cfg.FilePath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("Creating database directory: %s", dbDir)
		if err2 := os.MkdirAll(dbDir, 0o750); err2 != nil {
			log.Fatalf("Failed to create database directory '%s': %v", dbDir, err2)
		}
	} else if err != nil {
		log.Fatalf("Failed to check database directory '%s': %v", dbDir, err)
	}

	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 1
	}
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 1
	}
	connMaxLifetime := cfg.ConnMaxLifetime
	if cfg.FilePath == "" {
		return nil, fmt.Errorf("database.filepath cannot be empty in configuration")
	}
	dsn := fmt.Sprintf("%s?_journal=WAL&_busy_timeout=5000&_foreign_keys=on", cfg.FilePath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database at %s: %w", cfg.FilePath, err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database at %s: %w", cfg.FilePath, err)
	}
	fmt.Printf("SQLite database connection pool established for %s\n", cfg.FilePath)
	return db, nil
}
