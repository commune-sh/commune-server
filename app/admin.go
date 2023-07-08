package app

import (
	"context"
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

		err := c.Cache.System.Set("pinned", slug, 0).Err()
		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "Event not found.",
					"exists": false,
				},
			})
			return
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

		log.Println("pinnind event on index", slug)

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

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"unpinned": true,
			},
		})

	}
}
