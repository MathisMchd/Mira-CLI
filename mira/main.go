package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"mira/internal/notes"
	"mira/internal/search"
)

func usage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  mira add \"title\" \"content\"")
	fmt.Fprintln(out, "  mira list")
	fmt.Fprintln(out, "  mira search <query>")
	fmt.Fprintln(out, "  mira help")
}

func run(args []string, store notes.NoteStore, out io.Writer) int {
	if len(args) < 2 {
		usage(out)
		return 0
	}

	switch args[1] {
	case "add":
		if len(args) < 4 {
			fmt.Fprintln(out, "Usage: mira add \"title\" \"content\"")
			return 1
		}
		n := notes.NewNote(args[2], args[3])
		if err := store.Save(n); err != nil {
			fmt.Fprintf(out, "Erreur lors de la sauvegarde: %v\n", err)
			return 1
		}
		fmt.Fprintln(out, "Note ajoutée.")

	case "list":
		all, err := store.All()
		if err != nil {
			fmt.Fprintf(out, "Erreur lecture notes: %v\n", err)
			return 1
		}
		if len(all) == 0 {
			fmt.Fprintln(out, "Aucune note.")
			return 0
		}
		start := 0
		if len(all) > 10 {
			start = len(all) - 10
		}
		for i := len(all) - 1; i >= start; i-- {
			fmt.Fprintf(out, "- %s: %s\n", all[i].Title, all[i].Preview(80))
		}

	case "search":
		if len(args) < 3 {
			fmt.Fprintln(out, "Usage: mira search <query>")
			return 1
		}
		all, err := store.All()
		if err != nil {
			fmt.Fprintf(out, "Erreur lecture notes: %v\n", err)
			return 1
		}
		res := search.Search(all, strings.Join(args[2:], " "))
		if len(res) == 0 {
			fmt.Fprintln(out, "Aucun résultat.")
			return 0
		}
		for _, n := range res {
			fmt.Fprintf(out, "- %s: %s\n", n.Title, n.Preview(80))
		}

	case "help":
		usage(out)

	default:
		usage(out)
	}
	return 0
}

func main() {
	store, err := notes.NewJSONLStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur initialisation du magasin: %v\n", err)
		os.Exit(1)
	}
	os.Exit(run(os.Args, store, os.Stdout))
}
