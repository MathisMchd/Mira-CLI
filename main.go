package main

import (
	"fmt"
	"os"
	"strings"

	"mira/internal/notes"
	"mira/internal/search"
)

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  mira add \"title\" \"content\"")
	fmt.Println("  mira list")
	fmt.Println("  mira search <query>")
	fmt.Println("  mira help")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	cmd := os.Args[1]
	store, err := notes.NewJSONLStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur initialisation du magasin: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("Usage: mira add \"title\" \"content\"")
			os.Exit(1)
		}
		title := os.Args[2]
		content := os.Args[3]
		n := notes.NewNote(title, content)
		if err := store.Save(n); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur lors de la sauvegarde: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Note ajoutée.")

	case "list":
		all, err := store.All()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur lecture notes: %v\n", err)
			os.Exit(1)
		}
		count := len(all)
		if count == 0 {
			fmt.Println("Aucune note.")
			return
		}
		// show up to 10 newest notes
		start := 0
		if count > 10 {
			start = count - 10
		}
		for i := count - 1; i >= start; i-- {
			n := all[i]
			fmt.Printf("- %s: %s\n", n.Title, n.Preview(80))
		}

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mira search <query>")
			os.Exit(1)
		}
		q := strings.Join(os.Args[2:], " ")
		all, err := store.All()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur lecture notes: %v\n", err)
			os.Exit(1)
		}
		res := search.Search(all, q)
		if len(res) == 0 {
			fmt.Println("Aucun résultat.")
			return
		}
		for _, n := range res {
			fmt.Printf("- %s: %s\n", n.Title, n.Preview(80))
		}

	case "help":
		usage()

	default:
		usage()
	}
}
