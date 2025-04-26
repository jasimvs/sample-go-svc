// cmd/api/main.go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jasimvs/sample-go-svc/config"
	"github.com/jasimvs/sample-go-svc/internal/transaction"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// --- Configuration ---
	cfg, err := config.LoadConfig("./config") // Path relative to execution dir
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fmt.Printf("Configuration loaded: Port=%s\n", cfg.ServerPort)

	// --- Echo Instance ---
	e := echo.New()

	// --- Middleware ---
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	}))

	// --- Routes ---
	// Transaction routes handled by the transaction handler
	e.POST("/transactions", transaction.PostHandler)
	e.GET("/transactions", transaction.GetHandler)

	// Simple Ping endpoint (no specific handler needed)
	e.GET("/ping", func(c echo.Context) error {
		log.Println("Received /ping request")
		// Respond with 200 OK and no content in the body
		return c.NoContent(http.StatusOK)
	})

	// --- Start Server ---
	serverAddr := ":" + cfg.ServerPort
	log.Printf("Starting server on %s", serverAddr)
	if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
		e.Logger.Fatalf("Server shutdown error: %v", err)
	}
}
