package main

import (
	"log"

	"github.com/nb-322/SnorkProject/internal/client"
)

var serverAddr string

func main() {
	platformSetup()
	err := client.Connect(serverAddr)
	if err != nil {
		log.Fatal(err)
	}

}
