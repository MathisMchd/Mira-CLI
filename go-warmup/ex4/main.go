package main

import (
	"errors"
	"fmt"
	"sync"
)

var ErrValidation = errors.New("invalid note: title and content are required")
var ErrDuplicate = errors.New("note already exists")
var ErrNotFound = errors.New("note not found")

type Note struct {
	Title   string
	Content string
	Tags    []string
}

type NoteStore interface {
	Save(n *Note) error
	Get(title string) (*Note, error)
	All() []*Note
}

type MemoryStore struct {
	mutex sync.Mutex
	notes map[string]*Note
}

func (ms *MemoryStore) Save(note *Note) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	if note.Title == "" {
		return ErrValidation
	}

	if _, exists := ms.notes[note.Title]; exists {
		return ErrDuplicate
	}

	ms.notes[note.Title] = note
	return nil
}

func (ms *MemoryStore) Get(title string) (*Note, error) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	if n, exists := ms.notes[title]; exists {
		return n, nil
	}

	return nil, ErrNotFound
}

func (ms *MemoryStore) All() []*Note {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	notes := make([]*Note, 0, len(ms.notes))
	for _, note := range ms.notes {
		notes = append(notes, note)
	}
	return notes
}

func main() {
	store := &MemoryStore{
		notes: make(map[string]*Note),
	}

	fmt.Println("Bienvenue dans Mira, votre application de prise de notes !")
	fmt.Println("Commandes disponibles :")
	fmt.Println("- save <title> <content> : sauvegarder une note")
	fmt.Println("- get <title> : récupérer une note")
	fmt.Println("- all : afficher toutes les notes")
	fmt.Println("- exit : quitter l'application")
	fmt.Println("- help : afficher ce message d'aide")
	fmt.Println("Veuillez entrer une commande :")

	for {
		var command string
		fmt.Print("> ")
		fmt.Scan(&command)
		switch command {
		case "save":
			var title, content string
			fmt.Print("Titre : ")
			fmt.Scan(&title)
			fmt.Print("Contenu : ")
			fmt.Scan(&content)
			note := Note{
				Title:   title,
				Content: content,
				Tags:    []string{}, // faut que je le fasse mais il faut gérer un tableau de tags en plus dans la ligne de commande, je le ferai plus tard
			}
			if err := store.Save(&note); err != nil {
				if errors.Is(err, ErrValidation) {
					fmt.Println("Le titre et le contenu de la note sont requis.")
				} else if errors.Is(err, ErrDuplicate) {
					fmt.Println("Une note avec ce titre existe déjà.")
				}
				fmt.Printf("Erreur lors de la sauvegarde de la note : %v\n", err)
			} else {
				fmt.Println("Note sauvegardée avec succès !")
			}
		case "get":
			var title string
			fmt.Print("Titre : ")
			fmt.Scan(&title)
			note, err := store.Get(title)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					fmt.Println("Aucune note trouvée avec ce titre.")
				}
				fmt.Printf("Erreur lors de la récupération de la note : %v\n", err)
			} else {
				fmt.Printf("Titre : %s\nContenu : %s\nTags : %v\n", note.Title, note.Content, note.Tags)
			}
		case "all":
			notes := store.All()
			fmt.Printf("Nombre de notes : %d\n", len(notes))
			for _, note := range notes {
				fmt.Printf("Titre : %s\nContenu : %s\nTags : %v\n\n", note.Title, note.Content, note.Tags)
			}
		case "exit":
			fmt.Println("Au revoir !")
			return
		case "help":
			fmt.Println("Commandes disponibles :")
			fmt.Println("- save <title> <content> : sauvegarder une note")
			fmt.Println("- get <title> : récupérer une note")
			fmt.Println("- all : afficher toutes les notes")
			fmt.Println("- exit : quitter l'application")
			fmt.Println("- help : afficher ce message d'aide")
		default:
			fmt.Println("Commande inconnue. Veuillez utiliser 'save', 'get' ou 'all'.")
		}
	}
}
