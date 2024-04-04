package app

import (
	"context"
	"log"
	"net/http"
	matrix_db "commune/db/matrix/gen"

	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) Upvote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		event := query.Get("id")

		user := c.LoggedInUser(r)

		upvoted, err := c.MatrixDB.Queries.HasUpvoted(context.Background(), matrix_db.HasUpvotedParams{
			Sender: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			RelatesToID: event,
		})
		if err != nil {

			log.Println("error getting events: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		if !upvoted.Upvoted {

			_, err := c.NewPost(&NewPostParams{
				Body: &NewPostBody{
					Type:   "m.reaction",
					RoomID: upvoted.RoomID,
					Content: map[string]any{
						"m.relates_to": map[string]any{
							"rel_type": "m.annotation",
							"event_id": event,
							"key":      "upvote",
						},
					},
				},
				MatrixUserID:      user.MatrixUserID,
				MatrixAccessToken: user.MatrixAccessToken,
			})

			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error": "could not upvote",
					},
				})
				return
			}

		} else {
			log.Println("redacting upvote")
			_, err = c.RedactEvent(&RedactEventParams{
				RoomID:            upvoted.RoomID,
				EventID:           upvoted.EventID,
				MatrixUserID:      user.MatrixUserID,
				MatrixAccessToken: user.MatrixAccessToken,
			})

			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error": "could not redact upvote",
					},
				})
				return
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"upvoted": !upvoted.Upvoted,
			},
		})

	}
}

func (c *App) Downvote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		event := query.Get("id")

		user := c.LoggedInUser(r)

		downvoted, err := c.MatrixDB.Queries.HasDownvoted(context.Background(), matrix_db.HasDownvotedParams{
			Sender: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			RelatesToID: event,
		})

		if err != nil {
			log.Println("error getting events: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		if !downvoted.Downvoted {

			_, err := c.NewPost(&NewPostParams{
				Body: &NewPostBody{
					Type:   "m.reaction",
					RoomID: downvoted.RoomID,
					Content: map[string]any{
						"m.relates_to": map[string]any{
							"rel_type": "m.annotation",
							"event_id": event,
							"key":      "downvote",
						},
					},
				},
				MatrixUserID:      user.MatrixUserID,
				MatrixAccessToken: user.MatrixAccessToken,
			})

			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error": "could not downvote",
					},
				})
				return
			}

		} else {
			log.Println("redacting downvote")
			_, err = c.RedactEvent(&RedactEventParams{
				RoomID:            downvoted.RoomID,
				EventID:           downvoted.EventID,
				MatrixUserID:      user.MatrixUserID,
				MatrixAccessToken: user.MatrixAccessToken,
			})

			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error": "could not redact downvote",
					},
				})
				return
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"downvoted": !downvoted.Downvoted,
			},
		})

	}
}
