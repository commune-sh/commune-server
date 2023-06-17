package app

import (
	"log"
	"net/http"
)

type StateEventRequest struct {
	RoomID    string `json:"room_id"`
	EventType string `json:"event_type"`
	StateKey  string `json:"state_key"`
	Content   any    `json:"content"`
}

func (c *App) CreateStateEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &StateEventRequest{})

		if err != nil {
			log.Println(err)
			RespondWithBadRequestError(w)
			return
		}

		user := c.LoggedInUser(r)

		sse, err := c.NewStateEvent(&NewStateEventParams{
			RoomID:            p.RoomID,
			EventType:         p.EventType,
			StateKey:          p.StateKey,
			MatrixUserID:      user.MatrixUserID,
			MatrixAccessToken: user.MatrixAccessToken,
			Content:           p.Content,
		})

		if err != nil || sse == "" {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":   "could not create state event",
					"success": false,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success":  true,
				"event_id": sse,
			},
		})

	}
}
