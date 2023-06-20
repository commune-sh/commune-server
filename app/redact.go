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

type RedactEventParams struct {
	RoomID            string `json:"room_id"`
	EventID           string `json:"event_id"`
	Reason            string `json:"reason"`
	MatrixUserID      string `json:"matrix_user_id"`
	MatrixAccessToken string `json:"matrix_access_token"`
}

func (c *App) RedactEvent(p *RedactEventParams) (*gomatrix.RespSendEvent, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, p.MatrixUserID, p.MatrixAccessToken)
	if err != nil {
		return nil, err
	}

	resp, err := matrix.RedactEvent(p.RoomID, p.EventID, &gomatrix.ReqRedact{Reason: p.Reason})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *App) RedactPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			RoomID  string `json:"room_id"`
			EventID string `json:"event_id"`
			Reason  string `json:"reason"`
			IsReply bool   `json:"is_reply"`
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

		resp, err := c.RedactEvent(&RedactEventParams{
			RoomID:            p.RoomID,
			EventID:           p.EventID,
			Reason:            p.Reason,
			MatrixUserID:      user.MatrixUserID,
			MatrixAccessToken: user.MatrixAccessToken,
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

		if p.IsReply {
			go func() {
				_, err = c.MatrixDB.Exec(context.Background(), `REFRESH MATERIALIZED VIEW CONCURRENTLY reply_count`)
				if err != nil {
					log.Panicln(err)
				}
			}()
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

		resp, err := c.RedactEvent(&RedactEventParams{
			RoomID:            p.RoomID,
			EventID:           eventID,
			MatrixUserID:      user.MatrixUserID,
			MatrixAccessToken: user.MatrixAccessToken,
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
