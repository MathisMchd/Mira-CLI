package enrichment

import (
	"context"
	"fmt"
	"strings"

	"mira/internal/core"
)

const summaryMaxChars = 200

// OllamaEnricher délègue l'intégralité de l'enrichissement à des modèles
// Ollama locaux : un modèle génératif (OllamaGenerator) pour les tags, le
// résumé et le score, un modèle d'embedding (Embedder) pour le vecteur.
// 100% self-hosted, aucune clé API externe.
type OllamaEnricher struct {
	generator *OllamaGenerator
	embedder  Embedder
}

func NewOllamaEnricher(generator *OllamaGenerator, embedder Embedder) OllamaEnricher {
	return OllamaEnricher{generator: generator, embedder: embedder}
}

func (e OllamaEnricher) Enrich(ctx context.Context, note core.Note) (core.EnrichmentResult, error) {
	fields, err := e.generator.Generate(ctx, note.Title, note.Content)
	if err != nil {
		return core.EnrichmentResult{}, fmt.Errorf("génération tags/résumé/score: %w", err)
	}

	embedding, err := e.embedder.Embed(ctx, note.Title+" "+note.Content)
	if err != nil {
		return core.EnrichmentResult{}, fmt.Errorf("embedding: %w", err)
	}

	return core.EnrichmentResult{
		Tags:      mergeTags(note.Tags, fields.Tags),
		Summary:   truncateRunes(strings.TrimSpace(fields.Summary), summaryMaxChars),
		Score:     fields.Score,
		Embedding: embedding,
		Model:     e.embedder.Name(),
	}, nil
}

// mergeTags fusionne plusieurs listes de tags en dédupliquant (ordre
// préservé, première occurrence conservée).
func mergeTags(lists ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, list := range lists {
		for _, t := range list {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

// truncateRunes tronque au nombre de runes (pas d'octets, pour ne pas casser
// un caractère UTF-8 multi-octets comme les lettres accentuées).
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}
