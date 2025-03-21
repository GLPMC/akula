package main

import (
	"github.com/glpmc/akula/commands"
	"fmt"
	"os"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
