package main

import (
	"shpong/app"
	"flag"
	"log"
	"os"
	"os/signal"
)

func main() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)

	go func() {
		<-sc

		log.Println("Shutting down server")
		os.Exit(1)
	}()

	config := flag.String("config", "config.toml", "Shpong configuration file")

	flag.Parse()

	app.Start(&app.StartRequest{
		Config: *config,
	})
}
