package notes

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// JSONLStore stores notes in a JSON Lines file (~/.mira/notes.jsonl).
type JSONLStore struct {
	path  string
	mutex sync.Mutex
}

// NewJSONLStore creates (and ensures) the default store file and directory.
func NewJSONLStore() (*JSONLStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".mira")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	p := filepath.Join(dir, "notes.jsonl")
	// ensure file exists
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &JSONLStore{path: p}, nil
}

// Save appends a note as one JSON object per line.
func (s *JSONLStore) Save(n *Note) error {
	if n == nil || n.Title == "" || n.Content == "" {
		return errors.New("title and content required")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc, err := json.Marshal(n)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(enc, '\n')); err != nil {
		return err
	}
	return nil
}

// All reads all notes from the JSONL file in file order (oldest -> newest).
func (s *JSONLStore) All() ([]*Note, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	f, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var notesList []*Note
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var n Note
		if err := json.Unmarshal(line, &n); err != nil {
			// skip invalid line
			continue
		}
		notesList = append(notesList, &n)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return notesList, nil
}
