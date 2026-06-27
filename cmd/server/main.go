package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

}
