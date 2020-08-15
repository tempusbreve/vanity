package main

import (
	"log"

	"github.com/tempusbreve/vanity/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
