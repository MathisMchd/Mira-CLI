package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

type Note struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

func NewNote(title, content string) *Note {
	return &Note{
		Title:   title,
		Content: content,
		Tags:    []string{},
	}
}

func (n *Note) Preview(length int) string {
	if len(n.Content) <= length {
		return n.Content
	}

	return n.Content[:length]
}

func (n *Note) AddTag(tag string) {
	if !contains(n.Tags, tag) {
		n.Tags = append(n.Tags, tag)
	}
}

func contains(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func LoadFromFile(path string) ([]*Note, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Erreur lors de l'ouverture du fichier : %v", err)
	}

	jsonContent, err := ioutil.ReadAll(file)

	if err != nil {
		return nil, fmt.Errorf("Erreur lors de la lecture du fichier : %v", err)
	}
	file.Close()

	var notes []*Note
	err = json.Unmarshal(jsonContent, &notes)

	if err != nil {
		return nil, fmt.Errorf("Erreur lors de la désérialisation du JSON : %v", err)
	}
	return notes, nil
}

func main() {

	const lengthPreview = 80
	const fileName = "notes.json"

	fmt.Printf("Chargement des notes depuis le fichier : %s\n", fileName)
	notes, err := LoadFromFile(fileName)

	if err != nil {
		fmt.Printf("Erreur lors du chargement des notes : %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Notes chargées : %d\n", len(notes))

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Title < notes[j].Title
	})

	for _, note := range notes {
		fmt.Printf("Titre : %s\nAperçu : %s\nTags : %v\n\n", note.Title, note.Preview(lengthPreview), note.Tags)
	}

	notesWithGoTag := []string{}

	for _, note := range notes {
		if contains(note.Tags, "go") {
			notesWithGoTag = append(notesWithGoTag, note.Title)
		}
	}

	fmt.Printf("Notes avec le tag 'go' : %v\n", notesWithGoTag)

	fmt.Printf("Fin du chargement des notes.\n")

}
