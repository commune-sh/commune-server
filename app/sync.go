package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"commune/gomatrix"
)

func (c *App) SyncEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		token := query.Get("token")

		user, err := c.GetTokenUser(token)
		if err != nil || token == "" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("starting to sync for", user.Username)

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			log.Println(err)
		}

		//events := []any{}

		// Create a new channel to send events to the client
		eventCh := make(chan any)

		syncer := matrix.Syncer.(*gomatrix.DefaultSyncer)
		syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
			//fmt.Println("Message: ", ev)
			//events = append(events, ev)
			eventCh <- ev
		})

		go func() {
			for {
				if err := matrix.Sync(); err != nil {
					fmt.Println("Sync() returned ", err)
				}
				// Optional: Wait a period of time before trying to sync again.
			}
		}()

		log.Println("sending SSE to ", user.Username)

		// Set the content type to text/event-stream
		w.Header().Set("Content-Type", "text/event-stream")
		// Set cache-control header to prevent caching of the response
		w.Header().Set("Cache-Control", "no-cache")
		// Set connection header to keep the connection open
		w.Header().Set("Connection", "keep-alive")

		disconnect := make(chan bool)
		defer close(disconnect)

		// Continuously listen for events and write them to the response

		for {
			event := <-eventCh
			data, err := json.Marshal(event)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher, ok := w.(http.Flusher)
			if ok {
				flusher.Flush()
			}
			if disconnect != nil {
				eventCh = nil
			}
		}

	}
}

/*
func (c *App) SyncEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		log.Println("starting to sync for", user.Username)

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			log.Println(err)
		}

		events := []any{}

		syncer := matrix.Syncer.(*gomatrix.DefaultSyncer)
		syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
			fmt.Println("Message: ", ev)
			events = append(events, ev)
		})

		query := r.URL.Query()
		timeout := query.Get("timeout")

		go func() {
			for {
				if err := matrix.Sync(); err != nil {
					fmt.Println("Sync() returned ", err)
				}
				// Optional: Wait a period of time before trying to sync again.
			}
		}()

		if timeout != "" {
			log.Println("got since", timeout)
			duration, err := time.ParseDuration(timeout + "ms")
			if err != nil {
				fmt.Println("Error parsing duration:", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusInternalServerError,
					JSON: map[string]any{
						"error": "bad since value",
					},
				})
				return
			}

			time.Sleep(duration)
		}
		matrix.StopSync()

		// Set response headers for long-polling
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"syncing": true,
				"events":  events,
			},
		})

	}
}
*/
