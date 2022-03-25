package main

import (
	"log"

	"github.com/DagDigg/unpaper/backend/pkg/cmd"
	_ "github.com/lib/pq"
)

func main() {
	// Run server
	if err := cmd.RunServer(); err != nil {
		log.Fatalf("an error occurred while running the server: %v\n", err)
	}
}
