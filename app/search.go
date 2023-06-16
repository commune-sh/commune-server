package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-chi/chi/v5"
)

func (c *App) SearchEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		room_id := chi.URLParam(r, "room_id")

		query := r.URL.Query()
		q := query.Get("q")

		log.Println("searching: ", room_id, q)

		if q == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "no query provided",
				},
			})
			return
		}

		q = fmt.Sprintf("%s:*", q)

		events, err := c.MatrixDB.Queries.SearchEvents(context.Background(), q)

		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "no query provided",
				},
			})
			return
		}

		items := []Event{}

		for _, item := range events {

			json, err := gabs.ParseJSON([]byte(item.JSON.String))
			if err != nil {
				log.Println("error parsing json: ", err)
			}

			s := ProcessComplexEvent(&EventProcessor{
				EventID:     item.EventID,
				Slug:        item.Slug,
				RoomAlias:   item.RoomAlias.String,
				JSON:        json,
				DisplayName: item.DisplayName.String,
				AvatarURL:   item.AvatarUrl.String,
			})

			items = append(items, s)
		}
		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"results": items,
			},
		})

	}
}
