package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) SuspendUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		id := query.Get("id")

		log.Println("suspending user", id)

		user := c.LoggedInUser(r)

		if !user.Admin {

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":     "Not authorized.",
					"suspended": false,
				},
			})
			return
		}

		err := c.MatrixDB.Queries.DeactivateUser(context.Background(), pgtype.Text{
			String: id,
			Valid:  true,
		})
		if err != nil {
			log.Println("error deleting user", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":     "Error deleting user.",
					"suspended": false,
				},
			})
			return
		}

		err = c.PurgeUserSessions(id)
		if err != nil {
			log.Println("error deleting user", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":     "Error deleting user.",
					"suspended": false,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"suspended": true,
			},
		})

	}
}

func (c *App) PinEventToIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		slug := query.Get("slug")

		log.Println("pinnind event on index", slug)

		user := c.LoggedInUser(r)

		admin, err := c.MatrixDB.Queries.IsAdmin(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil || !admin {

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Not authorized.",
				},
			})
			return
		}

		pinned, err := c.Cache.System.Get("pinned").Result()
		if err != nil {
			list := []string{slug}
			serialized, err := json.Marshal(list)
			if err != nil {
				log.Println(err)
			}

			err = c.Cache.System.Set("pinned", serialized, 0).Err()
			if err != nil {
				log.Println("error getting event: ", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "Event could not be pinned.",
						"exists": false,
					},
				})
				return
			}

		} else {

			var us []string
			err = json.Unmarshal([]byte(pinned), &us)
			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "Event could not be pinned.",
						"exists": false,
					},
				})
				return
			}

			us = append(us, slug)

			serialized, err := json.Marshal(us)
			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "Event could not be pinned.",
						"exists": false,
					},
				})
				return
			}

			err = c.Cache.System.Set("pinned", serialized, 0).Err()
			if err != nil {
				log.Println("error getting event: ", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "Event could not be pinned.",
						"exists": false,
					},
				})
				return
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"pinned": true,
			},
		})

	}
}

func (c *App) UnpinIndexEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		slug := query.Get("slug")

		log.Println("unpinning event on index", slug)

		user := c.LoggedInUser(r)
		admin, err := c.MatrixDB.Queries.IsAdmin(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})

		if err != nil || !admin {

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Not authorized.",
				},
			})
			return
		}

		pinned, err := c.Cache.System.Get("pinned").Result()
		if err != nil {
			log.Println("error unpinning event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "Could not unpin.",
					"exists": false,
				},
			})
			return
		}
		var us []string
		err = json.Unmarshal([]byte(pinned), &us)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "Could not unpin.",
					"exists": false,
				},
			})
			return
		}

		if len(us) == 1 {
			err := c.Cache.System.Del("pinned").Err()
			if err != nil {
				log.Println("error unpinning event: ", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "Could not unpin.",
						"exists": false,
					},
				})
				return
			}
		}
		if len(us) > 1 {
			n := removeElement(us, slug)
			serialized, err := json.Marshal(n)
			if err != nil {
				log.Println(err)
			}

			err = c.Cache.System.Set("pinned", serialized, 0).Err()
			if err != nil {
				log.Println("error getting event: ", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "Event could not be pinned.",
						"exists": false,
					},
				})
				return
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"unpinned": true,
			},
		})

	}
}
