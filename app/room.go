package app

import (
	"context"
	"log"
	"net/http"

	matrix_db "shpong/db/matrix/gen"

	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) RoomJoined() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		room_id := query.Get("id")

		if room_id == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "internal server error",
				},
			})
			return
		}

		user := c.LoggedInUser(r)

		joined, err := c.MatrixDB.Queries.RoomJoined(context.Background(), matrix_db.RoomJoinedParams{
			UserID: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			RoomID: pgtype.Text{
				String: room_id,
				Valid:  true,
			},
		})

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "inter server error",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"joined": joined,
			},
		})

	}
}
