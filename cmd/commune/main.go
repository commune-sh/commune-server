package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"commune/app"
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

	req := &app.StartRequest{
		Config: *config,
	}

	if len(os.Args) > 1 {
		command := os.Args[1]

		switch command {
		case "views":
			req.MakeViews = true
		}
	}

	app.Start(req)
}
