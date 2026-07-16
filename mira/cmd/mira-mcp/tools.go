package main

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/internal/apiclient"
	"mira/internal/core"
)

// toolServer regroupe les dépendances partagées par les 4 handlers de tools :
// le client HTTP vers l'API mira (jamais le store en direct — c'est ce qui
// garantit que l'enrichissement automatique se déclenche), le logger (stderr
// uniquement) et le timeout appliqué à chaque appel API sous-jacent.
type toolServer struct {
	client  *apiclient.Client
	logger  *slog.Logger
	timeout time.Duration
}

// NoteSummary est une vue allégée d'une note, utilisée par search_notes et
// list_recent_notes : un extrait plutôt que le contenu complet, pour éviter de
// saturer le contexte de l'agent sur plusieurs résultats. L'id permet de
// récupérer le contenu complet via get_note.
type NoteSummary struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Snippet          string   `json:"snippet"`
	Summary          string   `json:"summary,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Score            float64  `json:"score,omitempty"`
	EnrichmentStatus string   `json:"enrichment_status"`
	CreatedAt        string   `json:"created_at"`
}

func toSummary(n *core.Note) NoteSummary {
	const snippetLen = 200
	snippet := n.Content
	if len(snippet) > snippetLen {
		snippet = snippet[:snippetLen]
	}
	return NoteSummary{
		ID:               n.ID,
		Title:            n.Title,
		Snippet:          snippet,
		Summary:          n.Summary,
		Tags:             n.Tags,
		Score:            n.Score,
		EnrichmentStatus: n.EnrichmentStatus,
		CreatedAt:        n.CreatedAt.Format(time.RFC3339),
	}
}

// clamp ramène v dans [min, max], et retombe sur def si v est absent (<=0).
func clamp(v, def, min, max int) int {
	if v <= 0 {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ---- search_notes ----

type SearchNotesInput struct {
	Query string `json:"query" jsonschema:"Texte de la recherche (mots-clés ou question en langage naturel). Recherche hybride combinant correspondance plein texte et similarité vectorielle sur le contenu et les résumés des notes."`
	Limit int    `json:"limit,omitempty" jsonschema:"Nombre maximum de notes à retourner (1 à 50). Par défaut 10."`
}

type SearchNotesOutput struct {
	Notes []NoteSummary `json:"notes" jsonschema:"Notes correspondantes, classées par pertinence."`
}

func (ts *toolServer) SearchNotes(ctx context.Context, _ *mcp.CallToolRequest, in SearchNotesInput) (*mcp.CallToolResult, SearchNotesOutput, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, SearchNotesOutput{}, fmt.Errorf("query est requis")
	}
	limit := clamp(in.Limit, 10, 1, 50)

	cctx, cancel := context.WithTimeout(ctx, ts.timeout)
	defer cancel()

	notes, err := ts.client.Search(cctx, query, limit)
	if err != nil {
		return nil, SearchNotesOutput{}, err
	}

	out := SearchNotesOutput{Notes: make([]NoteSummary, 0, len(notes))}
	for _, n := range notes {
		out.Notes = append(out.Notes, toSummary(n))
	}
	return nil, out, nil
}

// ---- get_note ----

type GetNoteInput struct {
	ID string `json:"id" jsonschema:"Identifiant unique (UUID) de la note à récupérer, tel que renvoyé par search_notes, add_note ou list_recent_notes."`
}

type GetNoteOutput struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	Tags             []string `json:"tags,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	EnrichmentStatus string   `json:"enrichment_status"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

func (ts *toolServer) GetNote(ctx context.Context, _ *mcp.CallToolRequest, in GetNoteInput) (*mcp.CallToolResult, GetNoteOutput, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, GetNoteOutput{}, fmt.Errorf("id est requis")
	}

	cctx, cancel := context.WithTimeout(ctx, ts.timeout)
	defer cancel()

	n, err := ts.client.GetByID(cctx, id)
	if err != nil {
		return nil, GetNoteOutput{}, err
	}

	return nil, GetNoteOutput{
		ID:               n.ID,
		Title:            n.Title,
		Content:          n.Content,
		Tags:             n.Tags,
		Summary:          n.Summary,
		EnrichmentStatus: n.EnrichmentStatus,
		CreatedAt:        n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        n.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ---- add_note ----

type AddNoteInput struct {
	Title   string   `json:"title" jsonschema:"Titre court et descriptif de la note (200 caractères maximum)."`
	Content string   `json:"content" jsonschema:"Contenu complet de la note en texte libre (markdown accepté)."`
	Tags    []string `json:"tags,omitempty" jsonschema:"Tags optionnels à associer à la note, en plus de ceux ajoutés automatiquement par l'enrichissement."`
}

type AddNoteOutput struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	EnrichmentStatus string `json:"enrichment_status" jsonschema:"Vaut 'pending' juste après la création : l'enrichissement (résumé, tags, embedding) tourne de façon asynchrone. Rappeler get_note plus tard pour vérifier qu'il est passé à 'done'."`
	CreatedAt        string `json:"created_at"`
}

func (ts *toolServer) AddNote(ctx context.Context, _ *mcp.CallToolRequest, in AddNoteInput) (*mcp.CallToolResult, AddNoteOutput, error) {
	input := core.CreateNoteInput{Title: in.Title, Content: in.Content, Tags: in.Tags}
	if err := input.Validate(); err != nil {
		return nil, AddNoteOutput{}, err
	}

	cctx, cancel := context.WithTimeout(ctx, ts.timeout)
	defer cancel()

	n, err := ts.client.Create(cctx, input)
	if err != nil {
		return nil, AddNoteOutput{}, err
	}

	return nil, AddNoteOutput{
		ID:               n.ID,
		Title:            n.Title,
		EnrichmentStatus: n.EnrichmentStatus,
		CreatedAt:        n.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ---- list_recent_notes ----

type ListRecentNotesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Nombre de notes à retourner (1 à 100), triées de la plus récente à la plus ancienne. Par défaut 10."`
}

type ListRecentNotesOutput struct {
	Notes []NoteSummary `json:"notes" jsonschema:"Notes les plus récemment créées, de la plus récente à la plus ancienne."`
}

func (ts *toolServer) ListRecentNotes(ctx context.Context, _ *mcp.CallToolRequest, in ListRecentNotesInput) (*mcp.CallToolResult, ListRecentNotesOutput, error) {
	limit := clamp(in.Limit, 10, 1, 100)

	cctx, cancel := context.WithTimeout(ctx, ts.timeout)
	defer cancel()

	notes, err := ts.client.List(cctx, limit, 0)
	if err != nil {
		return nil, ListRecentNotesOutput{}, err
	}

	out := ListRecentNotesOutput{Notes: make([]NoteSummary, 0, len(notes))}
	for _, n := range notes {
		out.Notes = append(out.Notes, toSummary(n))
	}
	return nil, out, nil
}

// withRecovery protège un handler de tool contre un panic imprévu : il est
// intercepté, loggé sur stderr avec sa pile, et transformé en erreur MCP
// propre plutôt que de faire planter tout le process (et donc la session
// stdio) ou de laisser fuiter une stack trace brute vers l'agent.
func withRecovery[In, Out any](logger *slog.Logger, name string, h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (res *mcp.CallToolResult, out Out, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered in tool handler",
					"tool", name, "error", r, "stack", string(debug.Stack()))
				err = fmt.Errorf("erreur interne lors de l'appel de %s", name)
			}
		}()
		return h(ctx, req, in)
	}
}
