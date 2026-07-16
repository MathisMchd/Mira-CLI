// Command mira-mcp expose la mémoire mira (notes, recherche hybride,
// enrichissement) à un agent IA via le Model Context Protocol, en transport
// stdio. Il passe systématiquement par l'API HTTP de mira (internal/apiclient),
// jamais par le store en direct : c'est le seul moyen de garantir que chaque
// note créée déclenche l'enrichissement automatique côté API.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/internal/apiclient"
)

func main() {
	// Le transport stdio réserve stdout au protocole JSON-RPC : tous les logs
	// doivent sortir sur stderr, jamais sur stdout.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("configuration invalide", "error", err)
		os.Exit(1)
	}

	client := apiclient.New(cfg.APIBaseURL, cfg.APIKey)
	ts := &toolServer{client: client, logger: logger, timeout: cfg.Timeout}

	server := mcp.NewServer(&mcp.Implementation{Name: "mira", Version: "0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "search_notes",
		Description: "Recherche hybride (plein texte + similarité vectorielle) dans les notes mira. " +
			"Retourne des résumés légers (titre, extrait, tags, statut d'enrichissement) classés par pertinence ; " +
			"utiliser get_note avec l'id retourné pour obtenir le contenu complet d'une note.",
	}, withRecovery(logger, "search_notes", ts.SearchNotes))

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_note",
		Description: "Récupère une note mira complète (contenu intégral, tags, résumé et statut d'enrichissement) " +
			"à partir de son id.",
	}, withRecovery(logger, "get_note", ts.GetNote))

	mcp.AddTool(server, &mcp.Tool{
		Name: "add_note",
		Description: "Crée une nouvelle note mira avec un titre et un contenu. Déclenche automatiquement " +
			"l'enrichissement asynchrone (résumé, tags, embedding) côté API : enrichment_status vaut 'pending' " +
			"juste après la création, puis passe à 'done' (ou 'failed') quelques secondes plus tard — " +
			"rappeler get_note pour vérifier.",
	}, withRecovery(logger, "add_note", ts.AddNote))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_recent_notes",
		Description: "Liste les notes mira les plus récemment créées, de la plus récente à la plus ancienne.",
	}, withRecovery(logger, "list_recent_notes", ts.ListRecentNotes))

	logger.Info("démarrage du serveur MCP mira", "api_url", cfg.APIBaseURL)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Error("le serveur MCP s'est arrêté en erreur", "error", err)
		os.Exit(1)
	}
}
