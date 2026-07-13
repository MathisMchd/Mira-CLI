package main

import (
	"fmt"
	"os"
)

func main() {

	const MaxDisplay = 10

	if len(os.Args) < 2 {
		fmt.Println("Usage : Entrer une liste de mots en argument.")
		fmt.Println("Exemple : go run main.go mot1 mot2 mot3 ...")
		os.Exit(1)

	} else if len(os.Args)-1 > MaxDisplay {

		fmt.Printf("Trop d'arguments. Affichage limité à %d.\n", MaxDisplay)

	} else {

		const wordLength = 4
		countWordsLengthSuperior := 0

		for _, arg := range os.Args[1:] {

			if len(arg) > wordLength {
				countWordsLengthSuperior++
			}
		}

		fmt.Printf(
			"Nombre de mots : %d.\nNombre de mots ayant une taille supérieure à %d : %d\n.",
			len(os.Args)-1,
			wordLength,
			countWordsLengthSuperior,
		)
	}
}
