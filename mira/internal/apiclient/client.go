// Package apiclient est le client HTTP utilisé par la CLI pour parler à
// l'API REST mira. La CLI ne touche plus jamais le stockage directement :
// c'est le seul moyen de garantir que chaque note créée ou modifiée déclenche
// l'enrichissement automatique (publié côté API après chaque POST/PATCH).
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mira/internal/core"
)

const defaultTimeout = 10 * time.Second

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// APIError représente une erreur explicite renvoyée par l'enveloppe JSON de
// l'API (champ "error").
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) Create(ctx context.Context, input core.CreateNoteInput) (*core.Note, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encodage requête: %w", err)
	}
	var n core.Note
	if err := c.do(ctx, http.MethodPost, "/api/v1/notes", body, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (c *Client) List(ctx context.Context, limit, offset int) ([]*core.Note, error) {
	path := fmt.Sprintf("/api/v1/notes?limit=%d&offset=%d", limit, offset)
	var notes []*core.Note
	if err := c.do(ctx, http.MethodGet, path, nil, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

func (c *Client) Search(ctx context.Context, query string) ([]*core.Note, error) {
	path := "/api/v1/search?q=" + url.QueryEscape(query)
	var notes []*core.Note
	if err := c.do(ctx, http.MethodGet, path, nil, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("construction requête: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("impossible de joindre l'API sur %s — vérifie qu'elle est démarrée (docker compose up): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("réponse API invalide (statut %d): %w", resp.StatusCode, err)
	}
	if env.Error != nil {
		return &APIError{Code: env.Error.Code, Message: env.Error.Message}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("l'API a répondu %d", resp.StatusCode)
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("décodage réponse: %w", err)
		}
	}
	return nil
}
