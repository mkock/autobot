// Copyright 2018 Martin Kock.

package main

import (
	"fmt"
	"log"

	"github.com/OmniCar/autobot/app"
)

func main() {
	if err := app.Start(); err != nil {
		log.Fatalf("Error: %s", err)
	}
	fmt.Println("Done.")
}
