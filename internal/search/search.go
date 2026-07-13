package search

import (
	"strings"

	"mira/internal/notes"
)

// Search performs a naive case-insensitive substring search on title+content.
func Search(list []*notes.Note, query string) []*notes.Note {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var res []*notes.Note
	for _, n := range list {
		if n == nil {
			continue
		}
		if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
			res = append(res, n)
		}
	}
	return res
}
