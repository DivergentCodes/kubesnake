// Package main is the entry point for the kubesnake CLI.
package main

import (
	"fmt"
	"os"

	"github.com/DivergentCodes/kubesnake/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
