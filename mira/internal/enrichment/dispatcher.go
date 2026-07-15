package enrichment

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"mira/internal/store"
)

// Job représente une note à enrichir.
type Job struct {
	NoteID string
}

// Dispatcher est un pool de workers borné consommant des jobs
// d'enrichissement depuis un channel interne bufferisé. L'envoi de jobs
// (Enqueue) est non-bloquant : si la file est pleine, le job est abandonné
// (log) plutôt que de retarder la réponse HTTP qui l'a déclenché — c'est ce
// qui garantit que l'écriture d'une note reste synchrone et rapide.
type Dispatcher struct {
	store    store.Store
	enricher Enricher
	logger   *slog.Logger
	timeout  time.Duration

	jobs chan Job
	wg   sync.WaitGroup
}

// NewDispatcher crée le pool et démarre immédiatement `workers` goroutines
// consommatrices. queueSize borne le nombre de jobs en attente ; timeout
// borne la durée de chaque job via context.WithTimeout.
func NewDispatcher(s store.Store, enricher Enricher, logger *slog.Logger, workers, queueSize int, timeout time.Duration) *Dispatcher {
	if workers <= 0 {
		workers = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}

	d := &Dispatcher{
		store:    s,
		enricher: enricher,
		logger:   logger,
		timeout:  timeout,
		jobs:     make(chan Job, queueSize),
	}

	d.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go d.worker()
	}
	return d
}

// Enqueue publie un job d'enrichissement de façon non-bloquante. Si la file
// est pleine, la note reste enrichment_status="pending" (pas de retry dans
// le scope de ce TP — limitation connue et assumée).
func (d *Dispatcher) Enqueue(noteID string) {
	select {
	case d.jobs <- Job{NoteID: noteID}:
	default:
		d.logger.Warn("file d'enrichissement pleine, job abandonné", "note_id", noteID)
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for job := range d.jobs {
		d.process(job)
	}
}

func (d *Dispatcher) process(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	note, err := d.store.GetByID(ctx, job.NoteID)
	if err != nil {
		d.logger.Error("enrichissement: note introuvable", "note_id", job.NoteID, "error", err)
		return
	}

	result, err := d.enricher.Enrich(ctx, *note)
	if err != nil {
		d.logger.Warn("enrichissement échoué", "note_id", job.NoteID, "error", err)
		d.markFailed(job.NoteID)
		return
	}

	// Contexte propre pour l'écriture : celui du job peut avoir expiré
	// juste après un Enrich qui a consommé tout le budget de temps.
	if err := d.store.SaveEnrichment(context.Background(), job.NoteID, result); err != nil {
		d.logger.Error("échec sauvegarde enrichissement", "note_id", job.NoteID, "error", err)
	}
}

func (d *Dispatcher) markFailed(noteID string) {
	if err := d.store.MarkEnrichmentFailed(context.Background(), noteID); err != nil {
		d.logger.Error("échec marquage enrichment_status=failed", "note_id", noteID, "error", err)
	}
}

// Stop ferme le channel de jobs et attend que les workers en cours terminent
// (borné par le timeout du job en cours). À appeler après l'arrêt du
// serveur HTTP pour ne perdre aucun job déjà accepté.
func (d *Dispatcher) Stop() {
	close(d.jobs)
	d.wg.Wait()
}
