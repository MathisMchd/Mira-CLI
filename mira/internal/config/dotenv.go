// Package config fournit un chargeur .env minimal, sans dépendance externe.
package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotEnv lit un fichier .env et positionne les variables d'environnement
// correspondantes, sans écraser celles déjà définies dans l'environnement.
// L'absence du fichier n'est pas une erreur.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}
