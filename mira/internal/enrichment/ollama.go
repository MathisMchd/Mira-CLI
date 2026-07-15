package enrichment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaEmbedder calcule des embeddings via un serveur Ollama local
// (endpoint /api/embeddings, ex. modèle "nomic-embed-text"). Aucune clé
// API : le modèle tourne dans son propre conteneur (voir docker-compose.yml,
// service "ollama"), 100% self-hosted.
type OllamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (o *OllamaEmbedder) Name() string { return "ollama:" + o.model }

type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (o *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(ollamaEmbedRequest{Model: o.model, Prompt: text})
	if err != nil {
		return nil, fmt.Errorf("encodage requête ollama: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("construction requête ollama: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("appel ollama: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("ollama a répondu %d: %s", resp.StatusCode, string(b))
	}

	var out ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("décodage réponse ollama: %w", err)
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("ollama a renvoyé un embedding vide")
	}
	return out.Embedding, nil
}

// OllamaGenerator produit tags, résumé et score via un modèle Ollama
// génératif (ex. "qwen2.5:1.5b-instruct"), en lui demandant une réponse JSON
// structurée (option "format":"json" de l'API Ollama, qui contraint le
// modèle à produire un JSON syntaxiquement valide).
type OllamaGenerator struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaGenerator(baseURL, model string) *OllamaGenerator {
	return &OllamaGenerator{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (g *OllamaGenerator) Name() string { return "ollama:" + g.model }

const generatePrompt = `Tu es un assistant qui enrichit des notes textuelles. Analyse la note ci-dessous et réponds UNIQUEMENT avec un objet JSON de cette forme exacte, sans aucun texte autour :
{"tags": ["mot-clé1", "mot-clé2"], "summary": "résumé en une phrase", "score": 0.5}

Règles :
- "tags" : 3 à 5 mots-clés pertinents, en minuscules, sans doublons.
- "summary" : résumé du contenu en une phrase, 200 caractères maximum.
- "score" : nombre entre 0 et 1 estimant la richesse/qualité du contenu.

Titre : %s
Contenu : %s`

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Format string `json:"format"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

// GeneratedFields est la sortie structurée demandée au modèle génératif.
type GeneratedFields struct {
	Tags    []string `json:"tags"`
	Summary string   `json:"summary"`
	Score   float64  `json:"score"`
}

func (g *OllamaGenerator) Generate(ctx context.Context, title, content string) (GeneratedFields, error) {
	prompt := fmt.Sprintf(generatePrompt, title, content)

	body, err := json.Marshal(ollamaGenerateRequest{Model: g.model, Prompt: prompt, Format: "json", Stream: false})
	if err != nil {
		return GeneratedFields{}, fmt.Errorf("encodage requête ollama: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return GeneratedFields{}, fmt.Errorf("construction requête ollama: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return GeneratedFields{}, fmt.Errorf("appel ollama: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return GeneratedFields{}, fmt.Errorf("ollama a répondu %d: %s", resp.StatusCode, string(b))
	}

	var out ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return GeneratedFields{}, fmt.Errorf("décodage réponse ollama: %w", err)
	}

	var fields GeneratedFields
	if err := json.Unmarshal([]byte(out.Response), &fields); err != nil {
		return GeneratedFields{}, fmt.Errorf("réponse ollama non-JSON (%q): %w", out.Response, err)
	}

	if fields.Score < 0 {
		fields.Score = 0
	} else if fields.Score > 1 {
		fields.Score = 1
	}
	return fields, nil
}
