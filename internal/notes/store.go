package notes

// NoteStore defines storage operations for notes.
type NoteStore interface {
	Save(n *Note) error
	All() ([]*Note, error)
}
