package main

import (
	"fmt"
	"sync"
)

func main() {
	compteur := 0
	var wg sync.WaitGroup
	var mu sync.Mutex // correction : protège compteur des accès concurrents

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			compteur++
			mu.Unlock()
		}()
	}

	wg.Wait()
	fmt.Println("Compteur final :", compteur)
}

// ---------------------------------------------------------------------
// Exercice 6 — Trouver et corriger une race condition
// ---------------------------------------------------------------------
//
// Version originale (sans mutex) :
//
//	go func() {
//	    defer wg.Done()
//	    compteur++
//	}()
//
// 1. Quel résultat obtenez-vous sans correction (exécutez plusieurs fois) ?
//
//    Le résultat affiché est presque toujours différent de 1000, et varie
//    d'une exécution à l'autre (par ex. 950, 987, 1000 parfois par chance,
//    971, ...). C'est parce que "compteur++" n'est PAS une opération
//    atomique : elle se décompose en 3 étapes (lire compteur, ajouter 1,
//    réécrire compteur). Quand deux goroutines exécutent ces étapes en
//    même temps, l'une peut écraser le résultat de l'autre : les deux
//    lisent la même valeur avant que l'une n'ait eu le temps d'écrire sa
//    mise à jour, et une incrémentation est donc "perdue".
//
// 2. Que rapporte `go run -race main.go` ?
//
//    Le détecteur de race signale un "DATA RACE" sur la variable
//    compteur : il indique qu'une goroutine est en train de lire
//    compteur (compteur++ lit sa valeur) pendant qu'une autre goroutine
//    est en train d'écrire dessus (compteur++ dans une autre goroutine),
//    sans mécanisme de synchronisation entre les deux accès. Il affiche
//    la pile d'appel des deux accès concurrents (write et read) pour
//    aider à localiser le problème dans le code.
//
// 3. Après correction avec `sync.Mutex`, le résultat est-il stable ?
//
//    Oui. Avec mu.Lock() / mu.Unlock() autour de compteur++, une seule
//    goroutine à la fois peut lire, incrémenter et réécrire compteur :
//    l'opération devient effectivement atomique du point de vue des
//    autres goroutines. Le résultat final est donc toujours exactement
//    1000, de façon stable et reproductible à chaque exécution, et
//    `go run -race main.go` ne rapporte plus aucune race condition.
