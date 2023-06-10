package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	matrix_db "shpong/db/matrix/gen"
	"shpong/gomatrix"

	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) RedactPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			RoomID  string `json:"room_id"`
			EventID string `json:"event_id"`
			Reason  string `json:"reason"`
		}{})

		if err != nil {
			log.Println(err)
			RespondWithBadRequestError(w)
			return
		}

		if p.RoomID == "" || p.EventID == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":    "no event id provided",
					"redacted": "false",
				},
			})
			return
		}

		user := c.LoggedInUser(r)

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":    err,
					"redacted": "false",
				},
			})
			return
		}

		resp, err := matrix.RedactEvent(p.RoomID, p.EventID, &gomatrix.ReqRedact{Reason: p.Reason})
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":    err,
					"redacted": "false",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"redacted": "true",
				"event":    resp.EventID,
			},
		})

	}
}

func (c *App) RedactReaction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			RoomID  string `json:"room_id"`
			EventID string `json:"event_id"`
			Key     string `json:"key"`
		}{})

		if err != nil {
			log.Println(err)
			RespondWithBadRequestError(w)
			return
		}

		if p.RoomID == "" || p.EventID == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":    "no event id provided",
					"redacted": "false",
				},
			})
			return
		}

		user := c.LoggedInUser(r)

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":    err,
					"redacted": "false",
				},
			})
			return
		}

		eventID, err := c.MatrixDB.Queries.GetReactionEventID(context.Background(), matrix_db.GetReactionEventIDParams{
			RoomID:      p.RoomID,
			RelatesToID: p.EventID,
			Sender: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			AggregationKey: pgtype.Text{
				String: p.Key,
				Valid:  true,
			},
		})
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":    err,
					"redacted": "false",
				},
			})
			return
		}

		resp, err := matrix.RedactEvent(p.RoomID, eventID, &gomatrix.ReqRedact{Reason: ""})
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":    err,
					"redacted": "false",
				},
			})
			return
		}

		log.Println(resp)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"redacted": "true",
				"event":    resp.EventID,
			},
		})

	}
}
