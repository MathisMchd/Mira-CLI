// Package enrichment fournit l'enrichissement automatique des notes (tags,
// résumé, score, embedding) et le pool de workers qui le déclenche de façon
// asynchrone après chaque création/modification de note.
package enrichment

import (
	"context"

	"mira/internal/core"
)

// Enricher calcule les enrichissements automatiques d'une note.
type Enricher interface {
	Enrich(ctx context.Context, note core.Note) (core.EnrichmentResult, error)
}

// Embedder calcule l'embedding vectoriel d'un texte — utilisé à la fois pour
// enrichir une note et pour vectoriser une requête de recherche. Implémenté
// par OllamaEmbedder (voir ollama.go), qui appelle un serveur Ollama local.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	// Name identifie le modèle utilisé (ex. "ollama:nomic-embed-text"),
	// enregistré à côté de chaque embedding en base pour traçabilité.
	Name() string
}
