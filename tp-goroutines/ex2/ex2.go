package main

import (
	"fmt"
	"sync"
	"time"
)

func afficherLettres(wg *sync.WaitGroup) {
	defer wg.Done()
	lettres := []string{"a", "b", "c", "d", "e"}
	for _, l := range lettres {
		fmt.Println(l)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres(wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 1; i <= 5; i++ {
		fmt.Println(i)
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	var wg sync.WaitGroup

	wg.Add(2)
	go afficherLettres(&wg)
	go afficherChiffres(&wg)

	// wg.Wait() bloque main() jusqu'à ce que les deux goroutines aient
	// appelé wg.Done() (via le defer pour être sur que c'est fait à la fin), sans jamais dépendre d'une durée
	// arbitraire comme le faisait time.Sleep dans l'exercice 1.
	wg.Wait()

	fmt.Println("Terminé.")
}
