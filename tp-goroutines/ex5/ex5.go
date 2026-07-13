package main

import (
	"fmt"
	"sync"
	"time"
)

const nbJobs = 20
const nbWorkers = 4

// worker lit des jobs depuis le channel jobs, calcule leur carré,
// et envoie le résultat dans le channel resultats.
// Un worker sur quatre (id == 1) simule un traitement lent en
// attendant 2 secondes avant de renvoyer chaque résultat.
func worker(id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		if id == 1 {
			time.Sleep(2 * time.Second)
		}
		resultats <- j * j
	}
}

func main() {
	jobs := make(chan int, nbJobs)
	resultats := make(chan int, nbJobs)
	var wg sync.WaitGroup

	for w := 1; w <= nbWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, resultats, &wg)
	}

	for j := 1; j <= nbJobs; j++ {
		jobs <- j
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultats)
	}()

	// Pour chaque job attendu, on attend soit un résultat sur le
	// channel resultats, soit l'expiration d'un délai de 500 ms via
	// time.After. Si le worker lent (id == 1, sleep 2s) n'a pas encore
	// répondu, on abandonne l'attente pour CE tour et on affiche un
	// message de timeout, sans bloquer le reste du programme. Le
	// résultat en retard n'est pas perdu : il reste dans le channel
	// (bufferisé) et sera récupéré lors d'un tour de boucle suivant.
	for i := 0; i < nbJobs; i++ {
		select {
		case r, ok := <-resultats:
			if !ok {
				break
			}
			fmt.Println("résultat reçu :", r)
		case <-time.After(500 * time.Millisecond):
			fmt.Println("timeout sur un résultat")
		}
	}
}
