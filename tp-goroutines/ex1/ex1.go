package main

import (
	"fmt"
	"time"
)

func afficherLettres() {
	lettres := []string{"a", "b", "c", "d", "e"}
	for _, l := range lettres {
		fmt.Println(l)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres() {
	for i := 1; i <= 5; i++ {
		fmt.Println(i)
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	go afficherLettres()
	afficherChiffres()

	// Question : que se passe-t-il si on retire ce time.Sleep final ?
	//
	// Réponse :
	// Le programme main() se termine dès que afficherChiffres() a fini
	// de s'exécuter dans la goroutine principale. Or en Go, quand la
	// fonction main() se termine, TOUTES les goroutines encore en cours
	// (y compris afficherLettres()) sont immédiatement tuées, qu'elles
	// aient fini leur travail ou non.
	// Sans le time.Sleep, il y a une "race" entre les deux goroutines :
	// - le plus souvent, main() se termine avant (ou pendant)
	//   l'exécution de afficherLettres(), donc on ne verra jamais
	//   (ou seulement partiellement) l'affichage des lettres a à e.
	// - le résultat n'est pas déterministe : il peut varier d'une
	//   exécution à l'autre.
	// C'est justement ce problème que le sync.WaitGroup de l'exercice 2
	// résout proprement, sans dépendre d'une pause arbitraire.
}
