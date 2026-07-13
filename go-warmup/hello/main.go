package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	fmt.Printf("Go %s sur %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if len(os.Args) > 1 {
		fmt.Printf("Bienvenue dans Mira, %s !\n", os.Args[1])
	} else {
		fmt.Printf("Usage :  go run main.go <votre prénom>. \nHello <votre prénom>. \n")
		os.Exit(1)
	}

}
