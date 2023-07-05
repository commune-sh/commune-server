package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	matrix_db "shpong/db/matrix/gen"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type SpaceStateParams struct {
	Slug         string
	MatrixUserID string
}

func (c *App) GetSpaceState(p *SpaceStateParams) (*SpaceState, error) {

	alias := c.ConstructMatrixRoomID(p.Slug)

	ssp := matrix_db.GetSpaceStateParams{
		RoomAlias: alias,
	}

	if p.MatrixUserID != "" {
		ssp.UserID = pgtype.Text{
			String: p.MatrixUserID,
			Valid:  true,
		}
	}

	state, err := c.MatrixDB.Queries.GetSpaceState(context.Background(), ssp)

	if err != nil {
		log.Println("error getting event: ", err)
		return nil, err
	}

	//hideRoom := state.IsPublic.Bool != state.Joined
	//log.Println("should we hide room? ", hideRoom)

	sps := ProcessState(state)

	return sps, nil
}

func (c *App) SpaceState() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		space := chi.URLParam(r, "space")

		space = strings.ToLower(space)

		isUser := strings.HasPrefix(space, "@")
		username := strings.TrimPrefix(space, "@")

		if isUser {

			deleted, err := c.MatrixDB.Queries.IsDeactivated(context.Background(), pgtype.Text{
				String: username,
				Valid:  true,
			})
			if err != nil || deleted {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "space does not exist",
						"exists": false,
					},
				})
				return
			}
		}

		ssp := SpaceStateParams{
			Slug: space,
		}

		if user != nil {
			ssp.MatrixUserID = user.MatrixUserID
		}

		state, err := c.GetSpaceState(&ssp)

		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "space does not exist",
					"exists": false,
				},
			})
			return
		}

		//log.Println("public, owner, joined", state.IsPublic, state.IsOwner, state.Joined)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"state": state,
			},
		})

	}
}

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

		log.Println(p)

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

func (c *App) GetPowerLevels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//user := c.LoggedInUser(r)

		space := chi.URLParam(r, "space")

		space = strings.ToLower(space)

		alias := c.ConstructMatrixRoomID(space)

		pl, err := c.MatrixDB.Queries.GetSpacePowerLevels(context.Background(), alias)

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "no power levels",
					"exists": false,
				},
			})
			return
		}
		var result map[string]interface{}
		err = json.Unmarshal(pl, &result)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"power_levels": result,
			},
		})

	}
}
