package main

import (
	"log"

	archive "github.com/ToshihitoKon/slack-archive"
)

func main() {
	if err := archive.Run(); err != nil {
		log.Fatal(err)
	}
}
