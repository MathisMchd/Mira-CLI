package store

import (
	"context"
	"sync"
	"testing"

	"mira/internal/core"
)

func TestMemoryStore_CreateAndGetByID(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	n, err := s.Create(ctx, core.CreateNoteInput{Title: "  Go  ", Content: "  contenu  ", Tags: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n.Title != "Go" || n.Content != "contenu" {
		t.Errorf("titre/contenu non nettoyés: %+v", n)
	}
	if n.EnrichmentStatus != core.EnrichmentPending {
		t.Errorf("enrichment_status = %q, attendu pending", n.EnrichmentStatus)
	}
	if n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
		t.Error("timestamps non renseignés")
	}

	got, err := s.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != n.ID || got.Title != n.Title {
		t.Errorf("GetByID = %+v, attendu correspondre à %+v", got, n)
	}

	// La copie retournée ne doit pas partager sa mémoire avec l'interne du
	// store : la muter ne doit rien changer côté store.
	got.Title = "modifié depuis l'extérieur"
	got2, _ := s.GetByID(ctx, n.ID)
	if got2.Title == "modifié depuis l'extérieur" {
		t.Error("GetByID a renvoyé un pointeur partagé avec l'état interne du store")
	}
}

func TestMemoryStore_GetByID_NotFound(t *testing.T) {
	s := NewMemory()
	_, err := s.GetByID(context.Background(), "does-not-exist")
	if !IsNotFound(err) {
		t.Errorf("err = %v, attendu ErrNotFound", err)
	}
}

func TestMemoryStore_List_orderAndPagination(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	var ids []string
	for i := 0; i < 3; i++ {
		n, err := s.Create(ctx, core.CreateNoteInput{Title: "n", Content: "c"})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		ids = append(ids, n.ID)
	}

	notes, total, err := s.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, attendu 3", total)
	}
	if len(notes) != 2 || notes[0].ID != ids[2] || notes[1].ID != ids[1] {
		t.Errorf("page 1 = %v, attendu les 2 plus récentes dans l'ordre [%s %s]", notes, ids[2], ids[1])
	}

	rest, total2, err := s.List(ctx, 2, 2)
	if err != nil {
		t.Fatalf("List offset: %v", err)
	}
	if total2 != 3 || len(rest) != 1 || rest[0].ID != ids[0] {
		t.Errorf("page 2 = %v (total %d), attendu 1 note %s", rest, total2, ids[0])
	}

	beyond, _, err := s.List(ctx, 10, 100)
	if err != nil {
		t.Fatalf("List offset hors limites: %v", err)
	}
	if len(beyond) != 0 {
		t.Errorf("offset hors limites: attendu liste vide, got %v", beyond)
	}
}

func TestMemoryStore_Patch(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	n, err := s.Create(ctx, core.CreateNoteInput{Title: "Titre", Content: "Contenu", Tags: []string{"a"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newContent := "Nouveau contenu"
	patched, err := s.Patch(ctx, n.ID, core.PatchNoteInput{Content: &newContent})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if patched.Title != "Titre" {
		t.Errorf("titre modifié alors qu'il n'était pas dans le patch: %q", patched.Title)
	}
	if patched.Content != newContent {
		t.Errorf("content = %q, attendu %q", patched.Content, newContent)
	}
	if len(patched.Tags) != 1 || patched.Tags[0] != "a" {
		t.Errorf("tags perdus alors qu'ils n'étaient pas dans le patch: %v", patched.Tags)
	}

	patched2, err := s.Patch(ctx, n.ID, core.PatchNoteInput{Tags: []string{"b", "c"}})
	if err != nil {
		t.Fatalf("patch tags: %v", err)
	}
	if len(patched2.Tags) != 2 {
		t.Errorf("tags = %v, attendu remplacés par [b c]", patched2.Tags)
	}
}

func TestMemoryStore_Patch_NotFound(t *testing.T) {
	s := NewMemory()
	title := "x"
	_, err := s.Patch(context.Background(), "does-not-exist", core.PatchNoteInput{Title: &title})
	if !IsNotFound(err) {
		t.Errorf("err = %v, attendu ErrNotFound", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	n, err := s.Create(ctx, core.CreateNoteInput{Title: "x", Content: "y"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.Delete(ctx, n.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetByID(ctx, n.ID); !IsNotFound(err) {
		t.Errorf("note toujours visible après delete: err = %v", err)
	}
	if err := s.Delete(ctx, n.ID); !IsNotFound(err) {
		t.Errorf("double delete: err = %v, attendu ErrNotFound", err)
	}

	// L'ordre d'insertion doit rester cohérent pour les notes restantes.
	n2, _ := s.Create(ctx, core.CreateNoteInput{Title: "reste", Content: "y"})
	notes, total, _ := s.List(ctx, 10, 0)
	if total != 1 || len(notes) != 1 || notes[0].ID != n2.ID {
		t.Errorf("état après delete incohérent: %v (total %d)", notes, total)
	}
}

func TestMemoryStore_Search(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	if _, err := s.Create(ctx, core.CreateNoteInput{Title: "Kubernetes", Content: "orchestration"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Create(ctx, core.CreateNoteInput{Title: "Recette", Content: "brioche maison"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	notes, err := s.Search(ctx, "KUBER", nil, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(notes) != 1 || notes[0].Title != "Kubernetes" {
		t.Errorf("résultats = %v, attendu 1 note 'Kubernetes' (recherche insensible à la casse)", notes)
	}

	none, err := s.Search(ctx, "inexistant", nil, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("attendu aucun résultat, got %v", none)
	}
}

func TestMemoryStore_SaveEnrichment(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	n, err := s.Create(ctx, core.CreateNoteInput{Title: "x", Content: "y"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = s.SaveEnrichment(ctx, n.ID, core.EnrichmentResult{
		Summary: "résumé", Tags: []string{"a"}, Score: 0.5,
	})
	if err != nil {
		t.Fatalf("SaveEnrichment: %v", err)
	}

	got, _ := s.GetByID(ctx, n.ID)
	if got.EnrichmentStatus != core.EnrichmentDone {
		t.Errorf("enrichment_status = %q, attendu done", got.EnrichmentStatus)
	}
	if got.Summary != "résumé" || got.Score != 0.5 {
		t.Errorf("summary/score = %q/%v", got.Summary, got.Score)
	}
}

func TestMemoryStore_SaveEnrichment_NotFound(t *testing.T) {
	s := NewMemory()
	err := s.SaveEnrichment(context.Background(), "does-not-exist", core.EnrichmentResult{})
	if !IsNotFound(err) {
		t.Errorf("err = %v, attendu ErrNotFound", err)
	}
}

func TestMemoryStore_MarkEnrichmentFailed(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	n, err := s.Create(ctx, core.CreateNoteInput{Title: "x", Content: "y"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.MarkEnrichmentFailed(ctx, n.ID); err != nil {
		t.Fatalf("MarkEnrichmentFailed: %v", err)
	}
	got, _ := s.GetByID(ctx, n.ID)
	if got.EnrichmentStatus != core.EnrichmentFailed {
		t.Errorf("enrichment_status = %q, attendu failed", got.EnrichmentStatus)
	}
}

func TestMemoryStore_MarkEnrichmentFailed_NotFound(t *testing.T) {
	s := NewMemory()
	err := s.MarkEnrichmentFailed(context.Background(), "does-not-exist")
	if !IsNotFound(err) {
		t.Errorf("err = %v, attendu ErrNotFound", err)
	}
}

// TestMemoryStore_ConcurrentAccess est un test de non-régression pour le bug
// de race détecté par `go test -race` en CI : GetByID copiait les champs
// d'une *core.Note partagée après avoir relâché le RWMutex, ce qui pouvait se
// chevaucher avec une écriture concurrente (ex. SaveEnrichment appelé par le
// worker d'enrichissement pendant qu'un handler HTTP lit la même note). Ce
// test lance des lectures et écritures concurrentes sur les mêmes notes ; il
// doit rester propre sous `-race`.
func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	const noteCount = 10
	ids := make([]string, noteCount)
	for i := 0; i < noteCount; i++ {
		n, err := s.Create(ctx, core.CreateNoteInput{Title: "n", Content: "c"})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		ids[i] = n.ID
	}

	var wg sync.WaitGroup
	for i := 0; i < noteCount; i++ {
		id := ids[i]

		wg.Add(3)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _ = s.GetByID(ctx, id)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = s.SaveEnrichment(ctx, id, core.EnrichmentResult{
					Summary: "résumé", Tags: []string{"t"}, Score: 1,
				})
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _, _ = s.List(ctx, 5, 0)
			}
		}()
	}
	wg.Wait()
}
