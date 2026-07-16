// Tests d'intégration exerçant le repository Postgres réel (migrations,
// transactions, recherche hybride full-text + pgvector) via un vrai
// conteneur PostgreSQL/pgvector démarré par testcontainers-go.
//
// Ils sont automatiquement ignorés (t.Skip) si Docker n'est pas disponible
// ou en mauvaise santé — go test ./... reste donc vert sur une machine sans
// Docker, tout en exerçant réellement le SQL en CI (Docker y est présent).
package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"mira/internal/core"
	"mira/internal/db"
	"mira/internal/store"
	"mira/internal/store/postgres"
)

var (
	poolOnce        sync.Once
	sharedPool      *pgxpool.Pool
	sharedContainer *tcpostgres.PostgresContainer
	setupErr        error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedContainer != nil {
		_ = sharedContainer.Terminate(context.Background())
	}
	os.Exit(code)
}

// testPool démarre, une seule fois pour tout le paquet, un conteneur
// Postgres+pgvector réel (même image que docker-compose.yml), y applique les
// migrations embarquées, et retourne un pool pgx prêt à l'emploi. Le
// conteneur est partagé entre tous les tests d'intégration pour rester
// rapide ; chaque test doit appeler truncateAll pour repartir d'une base vide.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	poolOnce.Do(func() {
		ctx := context.Background()
		c, err := tcpostgres.Run(ctx, "pgvector/pgvector:pg16",
			tcpostgres.WithDatabase("mira_test"),
			tcpostgres.WithUsername("mira"),
			tcpostgres.WithPassword("mira"),
			tcpostgres.BasicWaitStrategies(),
		)
		if err != nil {
			setupErr = fmt.Errorf("démarrage conteneur postgres: %w", err)
			return
		}
		sharedContainer = c

		dsn, err := c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			setupErr = fmt.Errorf("connection string: %w", err)
			return
		}
		if err := db.Migrate(dsn); err != nil {
			setupErr = fmt.Errorf("migrations: %w", err)
			return
		}
		pool, err := db.NewPool(ctx, dsn)
		if err != nil {
			setupErr = fmt.Errorf("pool pgx: %w", err)
			return
		}
		sharedPool = pool
	})
	if setupErr != nil {
		t.Fatalf("préparation base de test: %v", setupErr)
	}
	return sharedPool
}

// truncateAll vide toutes les tables entre deux tests pour garantir leur
// isolation malgré le conteneur/pool partagés entre tous les tests du paquet.
func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE notes, note_tags, note_embeddings RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func newRepo(t *testing.T) *postgres.Repository {
	pool := testPool(t)
	truncateAll(t, pool)
	return postgres.NewRepository(pool)
}

func TestRepository_CreateAndGetByID(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, core.CreateNoteInput{
		Title:   "  Go  ",
		Content: "  Un langage compilé  ",
		Tags:    []string{"go", "dev", "go"}, // doublon volontaire
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("ID vide après Create")
	}
	if created.Title != "Go" || created.Content != "Un langage compilé" {
		t.Errorf("titre/contenu non nettoyés: %+v", created)
	}
	if len(created.Tags) != 2 {
		t.Errorf("tags non dédupliqués: %v", created.Tags)
	}
	if created.EnrichmentStatus != core.EnrichmentPending {
		t.Errorf("enrichment_status = %q, attendu pending", created.EnrichmentStatus)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Error("timestamps non renseignés")
	}

	fetched, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched.Title != created.Title || fetched.Content != created.Content {
		t.Errorf("GetByID = %+v, attendu titre/contenu de %+v", fetched, created)
	}
	if len(fetched.Tags) != 2 || fetched.Tags[0] != "dev" || fetched.Tags[1] != "go" {
		t.Errorf("tags récupérés = %v, attendu [dev go] (triés)", fetched.Tags)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000"); !store.IsNotFound(err) {
		t.Errorf("UUID inexistant: err = %v, attendu ErrNotFound", err)
	}
	if _, err := repo.GetByID(ctx, "not-a-uuid"); !store.IsNotFound(err) {
		t.Errorf("UUID invalide: err = %v, attendu ErrNotFound (pas une erreur serveur brute)", err)
	}
}

func TestRepository_List_orderAndPagination(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	var ids []string
	for i := 0; i < 3; i++ {
		n, err := repo.Create(ctx, core.CreateNoteInput{Title: fmt.Sprintf("Note %d", i), Content: "..."})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		ids = append(ids, n.ID)
		time.Sleep(2 * time.Millisecond) // garantit des created_at distincts
	}

	notes, total, err := repo.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, attendu 3", total)
	}
	if len(notes) != 2 {
		t.Fatalf("len(notes) = %d, attendu 2", len(notes))
	}
	// La plus récente (dernière créée) en premier.
	if notes[0].ID != ids[2] || notes[1].ID != ids[1] {
		t.Errorf("ordre = [%s %s], attendu [%s %s]", notes[0].ID, notes[1].ID, ids[2], ids[1])
	}

	rest, total2, err := repo.List(ctx, 2, 2)
	if err != nil {
		t.Fatalf("List offset: %v", err)
	}
	if total2 != 3 || len(rest) != 1 || rest[0].ID != ids[0] {
		t.Errorf("page 2 = %+v (total %d), attendu 1 note %s", rest, total2, ids[0])
	}
}

func TestRepository_Patch_partialUpdateKeepsUntouchedFields(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	n, err := repo.Create(ctx, core.CreateNoteInput{Title: "Titre", Content: "Contenu", Tags: []string{"a"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newContent := "Nouveau contenu"
	patched, err := repo.Patch(ctx, n.ID, core.PatchNoteInput{Content: &newContent})
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

	patched2, err := repo.Patch(ctx, n.ID, core.PatchNoteInput{Tags: []string{"b", "c"}})
	if err != nil {
		t.Fatalf("patch tags: %v", err)
	}
	if len(patched2.Tags) != 2 || patched2.Tags[0] != "b" || patched2.Tags[1] != "c" {
		t.Errorf("tags après remplacement = %v, attendu [b c]", patched2.Tags)
	}
}

func TestRepository_Patch_NotFound(t *testing.T) {
	repo := newRepo(t)
	title := "x"
	_, err := repo.Patch(context.Background(), "00000000-0000-0000-0000-000000000000", core.PatchNoteInput{Title: &title})
	if !store.IsNotFound(err) {
		t.Errorf("err = %v, attendu ErrNotFound", err)
	}
}

func TestRepository_Delete(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	n, err := repo.Create(ctx, core.CreateNoteInput{Title: "x", Content: "y"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.Delete(ctx, n.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, n.ID); !store.IsNotFound(err) {
		t.Errorf("note toujours visible après delete: err = %v", err)
	}
	if err := repo.Delete(ctx, n.ID); !store.IsNotFound(err) {
		t.Errorf("double delete: err = %v, attendu ErrNotFound", err)
	}
}

func TestRepository_Search_fullTextMatchesOnlyRelevantNote(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if _, err := repo.Create(ctx, core.CreateNoteInput{Title: "Kubernetes", Content: "orchestration de conteneurs"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := repo.Create(ctx, core.CreateNoteInput{Title: "Recette", Content: "brioche maison"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	notes, err := repo.Search(ctx, "kubernetes", nil, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(notes) != 1 || notes[0].Title != "Kubernetes" {
		t.Errorf("résultats = %+v, attendu 1 note 'Kubernetes'", notes)
	}
}

func TestRepository_Search_limit(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if _, err := repo.Create(ctx, core.CreateNoteInput{Title: "Golang", Content: "notes sur golang"}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	notes, err := repo.Search(ctx, "golang", nil, 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("len(notes) = %d, attendu 2 (limit)", len(notes))
	}
}

// unitVector768 retourne un vecteur unitaire de dimension 768 avec un 1.0 à
// l'indice donné et des zéros ailleurs — pratique pour construire des paires
// de vecteurs à similarité cosinus connue (identiques -> 1.0, orthogonaux -> 0.0).
func unitVector768(axis int) []float32 {
	v := make([]float32, 768)
	v[axis] = 1
	return v
}

func TestRepository_Search_vectorSimilarityAboveThreshold(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	nearNote, err := repo.Create(ctx, core.CreateNoteInput{Title: "Note proche", Content: "contenu sans rapport avec la requête"})
	if err != nil {
		t.Fatalf("create near: %v", err)
	}
	farNote, err := repo.Create(ctx, core.CreateNoteInput{Title: "Note lointaine", Content: "autre contenu sans rapport"})
	if err != nil {
		t.Fatalf("create far: %v", err)
	}

	queryVec := unitVector768(0)
	if err := repo.SaveEnrichment(ctx, nearNote.ID, core.EnrichmentResult{Embedding: unitVector768(0), Model: "test"}); err != nil {
		t.Fatalf("save enrichment near: %v", err)
	}
	if err := repo.SaveEnrichment(ctx, farNote.ID, core.EnrichmentResult{Embedding: unitVector768(1), Model: "test"}); err != nil {
		t.Fatalf("save enrichment far: %v", err)
	}

	// Le mot-clé ne matche le plein texte d'aucune des deux notes : seule la
	// similarité vectorielle peut expliquer un résultat.
	notes, err := repo.Search(ctx, "xyzzy-terme-absent", queryVec, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != nearNote.ID {
		t.Errorf("résultats = %+v, attendu uniquement la note proche (%s)", notes, nearNote.ID)
	}
}

func TestRepository_SaveEnrichment(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	n, err := repo.Create(ctx, core.CreateNoteInput{Title: "x", Content: "y"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = repo.SaveEnrichment(ctx, n.ID, core.EnrichmentResult{
		Summary:   "résumé auto",
		Tags:      []string{"z", "a"},
		Score:     0.75,
		Embedding: unitVector768(5),
		Model:     "test-model",
	})
	if err != nil {
		t.Fatalf("SaveEnrichment: %v", err)
	}

	got, err := repo.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.EnrichmentStatus != core.EnrichmentDone {
		t.Errorf("enrichment_status = %q, attendu done", got.EnrichmentStatus)
	}
	if got.Summary != "résumé auto" || got.Score != 0.75 {
		t.Errorf("summary/score = %q/%v", got.Summary, got.Score)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "a" || got.Tags[1] != "z" {
		t.Errorf("tags = %v, attendu [a z]", got.Tags)
	}
}

func TestRepository_SaveEnrichment_NotFound(t *testing.T) {
	repo := newRepo(t)
	err := repo.SaveEnrichment(context.Background(), "00000000-0000-0000-0000-000000000000", core.EnrichmentResult{})
	if !store.IsNotFound(err) {
		t.Errorf("err = %v, attendu ErrNotFound", err)
	}
}

func TestRepository_MarkEnrichmentFailed(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	n, err := repo.Create(ctx, core.CreateNoteInput{Title: "x", Content: "y"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.MarkEnrichmentFailed(ctx, n.ID); err != nil {
		t.Fatalf("MarkEnrichmentFailed: %v", err)
	}

	got, err := repo.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.EnrichmentStatus != core.EnrichmentFailed {
		t.Errorf("enrichment_status = %q, attendu failed", got.EnrichmentStatus)
	}
	if got.Content != "y" {
		t.Errorf("contenu altéré par un échec d'enrichissement: %q", got.Content)
	}
}
