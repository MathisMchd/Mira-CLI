package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// callTimeout borne chaque appel individuel à l'API mira, via context.WithTimeout.
const callTimeout = 15 * time.Second

const defaultConfigPath = "config.json"

type config struct {
	APIBaseURL string
	APIKey     string
	Timeout    time.Duration
}

// fileConfig est le format de cmd/mira-mcp/config.json (voir config.example.json).
// Indépendant du fichier d'enregistrement d'un client MCP donné (.mcp.json) : ce
// fichier permet à mira-mcp de se configurer seul, quel que soit l'agent/hôte MCP
// qui le lance.
type fileConfig struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

// loadConfig applique, dans cet ordre de priorité croissante : les valeurs par
// défaut, le fichier JSON (s'il existe ; son absence n'est pas une erreur, même
// logique que internal/config.LoadDotEnv), puis les variables d'environnement
// MIRA_API_URL / MIRA_API_KEY qui l'emportent toujours si définies.
func loadConfig() (config, error) {
	cfg := config{
		APIBaseURL: "http://localhost:8080",
		Timeout:    callTimeout,
	}

	path := os.Getenv("MIRA_MCP_CONFIG")
	if path == "" {
		path = defaultConfigPath
	}

	fc, err := readFileConfig(path)
	if err != nil {
		return config{}, fmt.Errorf("lecture config %s: %w", path, err)
	}
	if fc != nil {
		if fc.APIURL != "" {
			cfg.APIBaseURL = fc.APIURL
		}
		if fc.APIKey != "" {
			cfg.APIKey = fc.APIKey
		}
	}

	if v := os.Getenv("MIRA_API_URL"); v != "" {
		cfg.APIBaseURL = v
	}
	if v := os.Getenv("MIRA_API_KEY"); v != "" {
		cfg.APIKey = v
	}

	return cfg, nil
}

func readFileConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("JSON invalide: %w", err)
	}
	return &fc, nil
}
