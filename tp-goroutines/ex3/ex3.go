package main

import "fmt"

const n = 1000
const nbGoroutines = 4

// sommePartielle calcule la somme des éléments de la tranche [debut:fin[
// de nombres, et envoie le résultat dans le channel resultat.
func sommePartielle(nombres []int, debut, fin int, resultat chan<- int) {
	somme := 0
	for _, v := range nombres[debut:fin] {
		somme += v
	}
	resultat <- somme
}

func main() {
	nombres := make([]int, n)
	for i := range nombres {
		nombres[i] = i + 1 // 1..1000
	}

	resultat := make(chan int)
	taille := n / nbGoroutines

	for i := 0; i < nbGoroutines; i++ {
		debut := i * taille
		fin := debut + taille
		go sommePartielle(nombres, debut, fin, resultat)
	}

	total := 0
	for i := 0; i < nbGoroutines; i++ {
		total += <-resultat
	}

	attendu := n * (n + 1) / 2
	fmt.Println("Somme calculée :", total)
	fmt.Println("Somme attendue :", attendu)
	fmt.Println("OK ?", total == attendu)
}
