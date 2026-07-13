package core

import (
	"fmt"
	"strings"
	"time"
)

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateNoteInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

func (i CreateNoteInput) Validate() error {
	var errs []string
	if strings.TrimSpace(i.Title) == "" {
		errs = append(errs, "title is required")
	}
	if len(i.Title) > 200 {
		errs = append(errs, "title must be 200 characters or fewer")
	}
	if strings.TrimSpace(i.Content) == "" {
		errs = append(errs, "content is required")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

type PatchNoteInput struct {
	Title   *string  `json:"title"`
	Content *string  `json:"content"`
	Tags    []string `json:"tags"`
}

func (i PatchNoteInput) Validate() error {
	var errs []string
	if i.Title != nil && strings.TrimSpace(*i.Title) == "" {
		errs = append(errs, "title cannot be empty")
	}
	if i.Title != nil && len(*i.Title) > 200 {
		errs = append(errs, "title must be 200 characters or fewer")
	}
	if i.Content != nil && strings.TrimSpace(*i.Content) == "" {
		errs = append(errs, "content cannot be empty")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
