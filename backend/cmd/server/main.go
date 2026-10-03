package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	_ "uptime-backend/docs"
	"uptime-backend/internal/config"
	"uptime-backend/internal/database"
	"uptime-backend/internal/server"
	"uptime-backend/internal/user"
)

// @title Uptime API
// @version 1.0
// @description HTTP API for managing user accounts, profiles, and uptime monitors.
// @BasePath /
// @schemes http
// @accept json
// @produce json
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Use a bearer access token: `Bearer {accessToken}`.
//
//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init --generalInfo main.go --dir .,../../internal/server --output ../../docs --parseInternal
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}
	store := user.NewStore(pool)
	handler, err := server.New(cfg, store, store)
	if err != nil {
		log.Fatal(err)
	}
	httpServer := &http.Server{Addr: cfg.Address, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	log.Printf("server listening on %s", cfg.Address)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
