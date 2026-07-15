package core

// EnrichmentResult est le produit de l'enrichissement automatique d'une note :
// tags suggérés, résumé court, score heuristique et embedding vectoriel.
// Model identifie le fournisseur d'embedding utilisé (ex. "ollama:nomic-embed-text"),
// enregistré à côté de l'embedding pour traçabilité.
type EnrichmentResult struct {
	Tags      []string
	Summary   string
	Score     float64
	Embedding []float32
	Model     string
}
