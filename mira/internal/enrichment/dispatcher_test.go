package enrichment_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"mira/internal/core"
	"mira/internal/enrichment"
	"mira/internal/store"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeEnricher permet de contrôler précisément le déroulement des jobs dans
// les tests : chaque appel à Enrich signale son démarrage sur `started` (note
// id) avant de bloquer jusqu'à ce que `release` soit fermé.
type fakeEnricher struct {
	started chan string
	release chan struct{}
	fail    bool
}

func newFakeEnricher() *fakeEnricher {
	return &fakeEnricher{started: make(chan string, 8), release: make(chan struct{})}
}

func (f *fakeEnricher) Enrich(ctx context.Context, note core.Note) (core.EnrichmentResult, error) {
	f.started <- note.ID
	<-f.release
	if f.fail {
		return core.EnrichmentResult{}, errBoom
	}
	return core.EnrichmentResult{
		Summary: "résumé de " + note.Title,
		Tags:    []string{"auto"},
		Score:   0.9,
	}, nil
}

var errBoom = errors.New("boom")

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("condition non atteinte après %s", timeout)
}

func TestDispatcher_processesEnqueuedJob(t *testing.T) {
	s := store.NewMemory()
	n, err := s.Create(context.Background(), core.CreateNoteInput{Title: "Go", Content: "..."})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	enricher := newFakeEnricher()
	d := enrichment.NewDispatcher(s, enricher, discardLogger(), 2, 4, time.Second)
	defer d.Stop()

	d.Enqueue(n.ID)
	<-enricher.started
	close(enricher.release)

	waitFor(t, time.Second, func() bool {
		got, _ := s.GetByID(context.Background(), n.ID)
		return got.EnrichmentStatus == core.EnrichmentDone
	})

	got, _ := s.GetByID(context.Background(), n.ID)
	if got.Summary == "" || len(got.Tags) == 0 {
		t.Errorf("note pas enrichie correctement: %+v", got)
	}
}

func TestDispatcher_enrichFailure_marksFailed(t *testing.T) {
	s := store.NewMemory()
	n, err := s.Create(context.Background(), core.CreateNoteInput{Title: "Go", Content: "..."})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	enricher := newFakeEnricher()
	enricher.fail = true
	d := enrichment.NewDispatcher(s, enricher, discardLogger(), 1, 1, time.Second)
	defer d.Stop()

	d.Enqueue(n.ID)
	<-enricher.started
	close(enricher.release)

	waitFor(t, time.Second, func() bool {
		got, _ := s.GetByID(context.Background(), n.ID)
		return got.EnrichmentStatus == core.EnrichmentFailed
	})
}

// TestDispatcher_boundedQueueDropsWhenFull vérifie que la file est bornée :
// avec workers=1 et queueSize=1, un 3e job soumis pendant que le worker est
// occupé et que la file est déjà pleine est abandonné plutôt que de bloquer
// l'appelant (Enqueue est non-bloquant par design).
func TestDispatcher_boundedQueueDropsWhenFull(t *testing.T) {
	s := store.NewMemory()
	ids := make([]string, 3)
	for i := range ids {
		n, err := s.Create(context.Background(), core.CreateNoteInput{Title: "N", Content: "..."})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		ids[i] = n.ID
	}

	enricher := newFakeEnricher()
	d := enrichment.NewDispatcher(s, enricher, discardLogger(), 1, 1, time.Second)
	defer d.Stop()

	d.Enqueue(ids[0])
	if got := <-enricher.started; got != ids[0] {
		t.Fatalf("attendu démarrage sur %s, got %s", ids[0], got)
	}
	// Le worker unique est maintenant bloqué dans Enrich(ids[0]) : la file est
	// vide et peut accepter un job.
	d.Enqueue(ids[1])
	// La file (capacité 1) contient déjà ids[1] et le worker est occupé :
	// ce 3e job doit être abandonné, pas mis en attente indéfiniment.
	d.Enqueue(ids[2])

	close(enricher.release)

	waitFor(t, time.Second, func() bool {
		a, _ := s.GetByID(context.Background(), ids[0])
		b, _ := s.GetByID(context.Background(), ids[1])
		return a.EnrichmentStatus == core.EnrichmentDone && b.EnrichmentStatus == core.EnrichmentDone
	})

	c, _ := s.GetByID(context.Background(), ids[2])
	if c.EnrichmentStatus != core.EnrichmentPending {
		t.Errorf("job 3 aurait dû être abandonné (rester pending), status = %s", c.EnrichmentStatus)
	}
}

// TestDispatcher_stopWaitsForInFlightJob vérifie que Stop() attend la fin des
// jobs déjà acceptés avant de rendre la main (arrêt propre, aucun job perdu).
func TestDispatcher_stopWaitsForInFlightJob(t *testing.T) {
	s := store.NewMemory()
	n, err := s.Create(context.Background(), core.CreateNoteInput{Title: "Go", Content: "..."})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	enricher := newFakeEnricher()
	d := enrichment.NewDispatcher(s, enricher, discardLogger(), 1, 1, time.Second)

	d.Enqueue(n.ID)
	<-enricher.started

	stopped := make(chan struct{})
	go func() {
		d.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("Stop() est revenu avant la fin du job en cours")
	case <-time.After(30 * time.Millisecond):
	}

	close(enricher.release)

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop() n'est jamais revenu après la fin du job")
	}

	got, _ := s.GetByID(context.Background(), n.ID)
	if got.EnrichmentStatus != core.EnrichmentDone {
		t.Errorf("job en cours perdu à l'arrêt: status = %s", got.EnrichmentStatus)
	}
}

func TestDispatcher_enqueueMissingNote_isNoOp(t *testing.T) {
	s := store.NewMemory()
	enricher := newFakeEnricher()
	d := enrichment.NewDispatcher(s, enricher, discardLogger(), 1, 1, time.Second)
	defer d.Stop()

	// La note n'existe pas : le worker doit logger et passer au job suivant
	// sans jamais appeler Enrich (donc sans bloquer sur `started`).
	d.Enqueue("does-not-exist")

	select {
	case id := <-enricher.started:
		t.Fatalf("Enrich n'aurait pas dû être appelé, got %s", id)
	case <-time.After(50 * time.Millisecond):
	}
}
