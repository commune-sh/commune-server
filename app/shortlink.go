package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (c *App) RedirectHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, c.Config.App.PublicDomain, http.StatusFound)
	}
}

func (c *App) ResolveShortlink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		event := chi.URLParam(r, "event")

		if event == "" {
			//redirect to home
			http.Redirect(w, r, c.Config.App.Domain, http.StatusFound)
			return
		}

		item, err := c.MatrixDB.Queries.GetShortlinkEvent(context.Background(), event)

		if err != nil || item.EventID == "" {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "event not found",
					"exists": false,
				},
			})
			return
		}

		slug := item.EventID[len(item.EventID)-11:]

		path := fmt.Sprintf("post/%s?view=board", slug)

		if item.Type == "m.room.message" {
			path = fmt.Sprintf("?view=chat&context=%s", slug)
		}

		url := fmt.Sprintf("%s/%s/%s", c.Config.App.PublicDomain, item.RoomAlias, path)

		http.Redirect(w, r, url, http.StatusFound)

	}
}
