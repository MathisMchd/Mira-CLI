package notes

import "time"

// Note represents a single note.
type Note struct {
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// NewNote constructs a new Note with the current timestamp.
func NewNote(title, content string) *Note {
	return &Note{
		Title:     title,
		Content:   content,
		Tags:      []string{},
		CreatedAt: time.Now(),
	}
}

// Preview returns the first `length` bytes of the content.
func (n *Note) Preview(length int) string {
	if length <= 0 {
		return ""
	}
	if len(n.Content) <= length {
		return n.Content
	}
	return n.Content[:length]
}
