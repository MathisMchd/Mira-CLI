package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"mira/internal/config"
	"mira/internal/db"
	"mira/internal/enrichment"
	apihttp "mira/internal/http"
	"mira/internal/store"
	"mira/internal/store/postgres"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// connectWithRetry applique les migrations puis ouvre le pool pgx, avec
// quelques tentatives espacées : utile si l'API démarre avant que Postgres
// ait fini son propre démarrage (le healthcheck docker-compose limite déjà
// ce risque, mais reste utile hors Docker).
func connectWithRetry(ctx context.Context, logger *slog.Logger, dsn string) (*pgxpool.Pool, error) {
	const maxAttempts = 10
	const delay = 2 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := db.Migrate(dsn); err != nil {
			lastErr = err
			logger.Warn("migration impossible, nouvelle tentative", "attempt", attempt, "error", err)
			time.Sleep(delay)
			continue
		}
		pool, err := db.NewPool(ctx, dsn)
		if err != nil {
			lastErr = err
			logger.Warn("connexion base impossible, nouvelle tentative", "attempt", attempt, "error", err)
			time.Sleep(delay)
			continue
		}
		return pool, nil
	}
	return nil, fmt.Errorf("échec de connexion après %d tentatives: %w", maxAttempts, lastErr)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := config.LoadDotEnv(".env"); err != nil {
		logger.Error("échec de lecture du .env", "error", err)
		os.Exit(1)
	}

	addr := getenv("ADDR", ":8080")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		logger.Error("DATABASE_URL doit être défini")
		os.Exit(1)
	}
	ollamaURL := getenv("OLLAMA_URL", "http://localhost:11434")
	ollamaEmbedModel := getenv("OLLAMA_EMBED_MODEL", "nomic-embed-text")
	ollamaGenModel := getenv("OLLAMA_GEN_MODEL", "qwen2.5:1.5b-instruct")
	workers := getenvInt("ENRICHMENT_WORKERS", 4)
	queueSize := getenvInt("ENRICHMENT_QUEUE_SIZE", 256)
	// Génération LLM sur CPU : nettement plus lent qu'un simple calcul
	// d'embedding, d'où un timeout par défaut plus généreux.
	enrichTimeout := getenvDuration("ENRICHMENT_TIMEOUT", 30*time.Second)

	ctx := context.Background()

	pool, err := connectWithRetry(ctx, logger, dsn)
	if err != nil {
		logger.Error("impossible de se connecter à PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connecté à PostgreSQL, migrations appliquées")

	repo := postgres.NewRepository(pool)

	embedder := enrichment.NewOllamaEmbedder(ollamaURL, ollamaEmbedModel)
	generator := enrichment.NewOllamaGenerator(ollamaURL, ollamaGenModel)
	enricher := enrichment.NewOllamaEnricher(generator, embedder)
	dispatcher := enrichment.NewDispatcher(repo, enricher, logger, workers, queueSize, enrichTimeout)

	var s store.Store = repo

	if os.Getenv("SEED") != "false" {
		if err := store.Seed(ctx, s, dispatcher.Enqueue); err != nil {
			logger.Error("échec du seed de démonstration", "error", err)
			os.Exit(1)
		}
		logger.Info("données de démonstration insérées")
	}

	handler := apihttp.NewRouter(s, dispatcher, embedder, logger)

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
		logger.Info("serveur démarré", "addr", addr, "enrichment_workers", workers)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("erreur fatale du serveur", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("signal reçu, arrêt en cours...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("arrêt forcé du serveur HTTP", "error", err)
	}

	dispatcher.Stop()
	logger.Info("serveur arrêté proprement")
}
