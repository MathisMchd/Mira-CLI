package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mira/internal/config"
	apihttp "mira/internal/http"
	"mira/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := config.LoadDotEnv(".env"); err != nil {
		logger.Error("échec de lecture du .env", "error", err)
		os.Exit(1)
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	s := store.NewMemory()

	if os.Getenv("SEED") != "false" {
		if err := store.Seed(s); err != nil {
			logger.Error("échec du seed de démonstration", "error", err)
			os.Exit(1)
		}
		logger.Info("données de démonstration insérées")
	}

	handler := apihttp.NewRouter(s, logger)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("impossible de démarrer le serveur", "addr", addr, "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("serveur démarré", "addr", addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("erreur fatale du serveur", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("signal reçu, arrêt en cours...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("arrêt forcé", "error", err)
		os.Exit(1)
	}
	logger.Info("serveur arrêté proprement")
}
