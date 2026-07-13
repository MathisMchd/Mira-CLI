package main

import (
	"fmt"
	"sync"
)

const nbJobs = 20
const nbWorkers = 4

// worker lit des jobs depuis le channel jobs, calcule leur carré,
// et envoie le résultat dans le channel resultats.
func worker(_ int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		resultats <- j * j
	}
}

func main() {
	jobs := make(chan int, nbJobs)
	resultats := make(chan int, nbJobs)
	var wg sync.WaitGroup

	// Démarrage des workers
	for w := 1; w <= nbWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, resultats, &wg)
	}

	// Envoi des jobs puis fermeture du channel jobs : les workers
	// s'arrêteront de boucler (for j := range jobs) une fois tous
	// les jobs consommés.
	for j := 1; j <= nbJobs; j++ {
		jobs <- j
	}
	close(jobs)

	// Une fois que tous les workers ont terminé (wg.Wait), on peut
	// fermer resultats en toute sécurité : plus personne n'y écrira.
	go func() {
		wg.Wait()
		close(resultats)
	}()

	for r := range resultats {
		fmt.Println(r)
	}

	// Question : pourquoi l'ordre des résultats n'est-il pas garanti ?
	//
	// Réponse :
	// Les 4 workers tournent en parallèle et lisent dans le même
	// channel "jobs" de façon concurrente : le scheduler Go décide de
	// façon non déterministe quel worker récupère quel job et quand il
	// a fini de le traiter. Un worker peut être ralenti (charge CPU,
	// scheduling, etc.) pendant qu'un autre traite plusieurs jobs plus
	// vite. Comme chaque worker écrit son résultat dans "resultats" dès
	// qu'il a fini, l'ordre d'arrivée dans ce channel dépend de l'ordre
	// réel de fin de traitement, pas de l'ordre d'envoi initial des
	// jobs. Il n'y a donc aucune garantie que le résultat de job 1
	// arrive avant celui de job 2, etc.
}
