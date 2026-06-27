package main

import (
	"log"

	"github.com/nb-322/SnorkProject/internal/server"
)

func main() {
	s, err := server.NewServer(4444)
	if err != nil {
		log.Fatalf("server.NewServer error: %s", err)
	}
	go s.Serve()
	cli := server.NewCLI(s)
	go cli.Run()
	select {}
}
