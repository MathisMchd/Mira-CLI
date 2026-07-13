package store

import (
	"errors"
	"fmt"

	"mira/internal/core"
)

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("note not found: %s", e.ID) }

func IsNotFound(err error) bool {
	var e ErrNotFound
	return errors.As(err, &e)
}

type Store interface {
	Create(input core.CreateNoteInput) (*core.Note, error)
	GetByID(id string) (*core.Note, error)
	List(limit, offset int) ([]*core.Note, int, error)
	Patch(id string, input core.PatchNoteInput) (*core.Note, error)
	Delete(id string) error
	Search(query string) ([]*core.Note, error)
}
