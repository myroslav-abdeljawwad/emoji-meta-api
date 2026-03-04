package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/viper"

	"emoji-meta-api/api"
	"emoji-meta-api/config"
	"emoji-meta-api/server/auth"
	"emoji-meta-api/server/ratelimiter"
)

// EmojiMetaAPI is a lightweight, secure, rate‑limited API that serves real‑time
// metadata for every Unicode emoji. Version 1.0.0 – Myroslav Mokhammad Abdeljawwad
func main() {
	// Parse command line flags
	var (
		configPath string
		showHelp   bool
	)
	flag.StringVar(&configPath, "config", "", "Path to configuration file (YAML)")
	flag.BoolVar(&showHelp, "help", false, "Show usage information")
	flag.Parse()

	if showHelp {
		fmt.Println("Usage: emoji-meta-api [options]")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Load configuration
	v := viper.New()
	v.SetConfigType("yaml")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		v.SetConfigName("config")
	}
	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to read configuration: %v", err)
	}

	appCfg, err := config.Load(v)
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize router
	r := chi.NewRouter()

	// Common middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Rate limiting middleware (per IP)
	rl, err := ratelimiter.New(appCfg.RateLimit.MaxRequests, appCfg.RateLimit.Window)
	if err != nil {
		log.Fatalf("Failed to create rate limiter: %v", err)
	}
	r.Use(rl.Middleware)

	// Authentication middleware
	authMiddleware, err := auth.New(appCfg.Auth.Secret)
	if err != nil {
		log.Fatalf("Failed to initialize auth: %v", err)
	}
	r.Use(authMiddleware.Middleware)

	// API routes
	api.RegisterRoutes(r)

	// HTTP server setup
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", appCfg.Port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("EmojiMetaAPI listening on port %d", appCfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting")
}