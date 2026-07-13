package main

import (
	"fmt"
	"os"
	"sort"
)

type tagCount struct {
	tag   string
	count int
}

func main() {
	// Récupère les tags
	tags := os.Args[1:]

	counts := make(map[string]int)

	for _, tag := range tags {
		counts[tag]++
	}

	var result []tagCount

	for tag, count := range counts {

		result = append(result, tagCount{
			tag:   tag,
			count: count,
		})

	}

	// Trie par fréquence décroissante
	sort.Slice(result, func(i, j int) bool {
		return result[i].count > result[j].count
	})

	// Affichage de tous les tags
	fmt.Println("Tags et leurs fréquences :")
	for _, item := range result {
		fmt.Printf("%s: %d\n", item.tag, item.count)
	}

	// Affichage des tags de fréquence supérieure à 1
	fmt.Println("\nTags avec une fréquence supérieure à 1 :")
	for _, item := range result {
		if item.count > 1 {
			fmt.Printf("%s: %d\n", item.tag, item.count)
		}
	}
}
