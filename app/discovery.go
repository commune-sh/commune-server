package app

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func (c *App) Handshake() {
	log.Println("initiating handshake with", c.Config.Discovery.Server)

	url := fmt.Sprintf("%s/handshake?domain=%s&key=%s", c.Config.Discovery.Server, c.Config.Discovery.Domain, c.Config.Discovery.Key)

	response, err := http.Get(url)
	if err != nil {
		log.Printf("Error making the request: %s\n", err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error reading the response: %s\n", err)
		return
	}

	log.Println(string(body))
}
