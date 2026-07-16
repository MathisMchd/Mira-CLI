package main

import (
	"os"
	"path/filepath"
	"testing"
)

// clearMCPEnv force à vide les variables lues par loadConfig : le code les
// traite comme "absentes" dès lors qu'elles valent "" (cf. loadConfig), donc
// t.Setenv(key, "") a le même effet qu'un unset pour ce test, tout en
// restaurant automatiquement la valeur d'origine à la fin du test.
func clearMCPEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MIRA_MCP_CONFIG", filepath.Join(t.TempDir(), "does-not-exist.json"))
	t.Setenv("MIRA_API_URL", "")
	t.Setenv("MIRA_API_KEY", "")
}

func TestLoadConfig_defaultsWhenNothingConfigured(t *testing.T) {
	clearMCPEnv(t)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.APIBaseURL != "http://localhost:8080" {
		t.Errorf("APIBaseURL = %q, attendu http://localhost:8080", cfg.APIBaseURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, attendu vide", cfg.APIKey)
	}
	if cfg.Timeout != callTimeout {
		t.Errorf("Timeout = %v, attendu %v", cfg.Timeout, callTimeout)
	}
}

func TestLoadConfig_fileOverridesDefaults(t *testing.T) {
	clearMCPEnv(t)

	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"api_url":"http://mira-file:9090","api_key":"file-key"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("écriture fichier de config: %v", err)
	}
	t.Setenv("MIRA_MCP_CONFIG", path)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.APIBaseURL != "http://mira-file:9090" {
		t.Errorf("APIBaseURL = %q, attendu la valeur du fichier", cfg.APIBaseURL)
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("APIKey = %q, attendu la valeur du fichier", cfg.APIKey)
	}
}

func TestLoadConfig_envOverridesFile(t *testing.T) {
	clearMCPEnv(t)

	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"api_url":"http://mira-file:9090","api_key":"file-key"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("écriture fichier de config: %v", err)
	}
	t.Setenv("MIRA_MCP_CONFIG", path)
	t.Setenv("MIRA_API_URL", "http://mira-env:7070")
	t.Setenv("MIRA_API_KEY", "env-key")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.APIBaseURL != "http://mira-env:7070" {
		t.Errorf("APIBaseURL = %q, attendu que l'env prime sur le fichier", cfg.APIBaseURL)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("APIKey = %q, attendu que l'env prime sur le fichier", cfg.APIKey)
	}
}

func TestLoadConfig_missingFileIsNotAnError(t *testing.T) {
	clearMCPEnv(t)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("fichier de config absent: err = %v, attendu nil", err)
	}
	if cfg.APIBaseURL != "http://localhost:8080" {
		t.Errorf("APIBaseURL = %q, attendu le défaut", cfg.APIBaseURL)
	}
}

func TestLoadConfig_invalidJSONReturnsError(t *testing.T) {
	clearMCPEnv(t)

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("écriture fichier de config: %v", err)
	}
	t.Setenv("MIRA_MCP_CONFIG", path)

	if _, err := loadConfig(); err == nil {
		t.Fatal("attendu une erreur pour un config.json invalide")
	}
}
