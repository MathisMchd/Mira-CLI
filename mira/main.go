package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"mira/internal/apiclient"
	"mira/internal/core"
)

func usage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "Usage:")
	_, _ = fmt.Fprintln(out, "  mira add \"title\" \"content\"")
	_, _ = fmt.Fprintln(out, "  mira list")
	_, _ = fmt.Fprintln(out, "  mira search <query>")
	_, _ = fmt.Fprintln(out, "  mira help")
}

func preview(content string, length int) string {
	if length <= 0 {
		return ""
	}
	if len(content) <= length {
		return content
	}
	return content[:length]
}

func run(args []string, client *apiclient.Client, out io.Writer) int {
	if len(args) < 2 {
		usage(out)
		return 0
	}

	ctx := context.Background()

	switch args[1] {
	case "add":
		if len(args) < 4 {
			_, _ = fmt.Fprintln(out, "Usage: mira add \"title\" \"content\"")
			return 1
		}
		input := core.CreateNoteInput{Title: args[2], Content: args[3]}
		if _, err := client.Create(ctx, input); err != nil {
			_, _ = fmt.Fprintf(out, "Erreur lors de la sauvegarde: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(out, "Note ajoutée.")

	case "list":
		notes, err := client.List(ctx, 10, 0)
		if err != nil {
			_, _ = fmt.Fprintf(out, "Erreur lecture notes: %v\n", err)
			return 1
		}
		if len(notes) == 0 {
			_, _ = fmt.Fprintln(out, "Aucune note.")
			return 0
		}
		for _, n := range notes {
			_, _ = fmt.Fprintf(out, "- %s: %s\n", n.Title, preview(n.Content, 80))
		}

	case "search":
		if len(args) < 3 {
			_, _ = fmt.Fprintln(out, "Usage: mira search <query>")
			return 1
		}
		notes, err := client.Search(ctx, strings.Join(args[2:], " "))
		if err != nil {
			_, _ = fmt.Fprintf(out, "Erreur recherche: %v\n", err)
			return 1
		}
		if len(notes) == 0 {
			_, _ = fmt.Fprintln(out, "Aucun résultat.")
			return 0
		}
		for _, n := range notes {
			_, _ = fmt.Fprintf(out, "- %s: %s\n", n.Title, preview(n.Content, 80))
		}

	case "help":
		usage(out)

	default:
		usage(out)
	}
	return 0
}

func main() {
	baseURL := os.Getenv("MIRA_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	client := apiclient.New(baseURL)
	os.Exit(run(os.Args, client, os.Stdout))
}
