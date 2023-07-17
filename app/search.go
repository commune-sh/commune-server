package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	matrix_db "shpong/db/matrix/gen"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) AddSearchEvent(e map[string]interface{}) {

	doc := []map[string]interface{}{e}

	log.Println("doc is", doc)
	up, err := c.SearchStore.Index("events").AddDocuments(doc)
	if err != nil {
		log.Println(err)
	}
	finalTask, err := c.SearchStore.Index("events").WaitForTask(up.TaskUID)
	if err != nil {
		log.Println(err)
	}
	if finalTask.Status != "succeeded" {
		log.Println(finalTask)
	}
}

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

		/*
			{
				searchRes, err := c.SearchStore.Index("events").Search(q,
					&meilisearch.SearchRequest{
						Limit: 10,
					})
				if err != nil {
					log.Println(err)
				}

				log.Println(searchRes.Hits)
			}
		*/

		events, err := c.MatrixDB.Queries.SearchEvents(context.Background(), matrix_db.SearchEventsParams{
			RoomID: pgtype.Text{
				String: room_id,
				Valid:  true,
			},
			Query: pgtype.Text{
				String: q,
				Valid:  true,
			},
			Wildcard: pgtype.Text{
				String: fmt.Sprintf("%s:*", q),
				Valid:  true,
			},
		})

		if err != nil {
			log.Println(err)
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
