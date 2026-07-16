package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv_setsUnsetVariables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "FOO=bar\nQUOTED=\"hello world\"\nSINGLE='single'\n\n# commentaire\nBAZ=qux\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("écriture fichier de test: %v", err)
	}

	for _, key := range []string{"FOO", "QUOTED", "SINGLE", "BAZ"} {
		os.Unsetenv(key)
		t.Cleanup(func(k string) func() { return func() { os.Unsetenv(k) } }(key))
	}

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	cases := map[string]string{
		"FOO":    "bar",
		"QUOTED": "hello world",
		"SINGLE": "single",
		"BAZ":    "qux",
	}
	for key, want := range cases {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, attendu %q", key, got, want)
		}
	}
}

func TestLoadDotEnv_doesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("PRIORITY=from-file\n"), 0o644); err != nil {
		t.Fatalf("écriture fichier de test: %v", err)
	}

	os.Setenv("PRIORITY", "from-env")
	t.Cleanup(func() { os.Unsetenv("PRIORITY") })

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("PRIORITY"); got != "from-env" {
		t.Errorf("PRIORITY = %q, l'environnement aurait dû primer sur le fichier", got)
	}
}

func TestLoadDotEnv_missingFileIsNotAnError(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), "does-not-exist.env")); err != nil {
		t.Errorf("fichier absent: err = %v, attendu nil", err)
	}
}

func TestLoadDotEnv_ignoresBlankLinesAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("\n# juste un commentaire\n   \nNOMALFORMED\nOK=1\n"), 0o644); err != nil {
		t.Fatalf("écriture fichier de test: %v", err)
	}
	os.Unsetenv("OK")
	os.Unsetenv("NOMALFORMED")
	t.Cleanup(func() { os.Unsetenv("OK") })

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if os.Getenv("OK") != "1" {
		t.Errorf("OK = %q, attendu 1", os.Getenv("OK"))
	}
	if _, ok := os.LookupEnv("NOMALFORMED"); ok {
		t.Error("une ligne sans '=' ne devrait définir aucune variable")
	}
}
