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
