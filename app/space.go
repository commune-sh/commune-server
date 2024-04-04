package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	matrix_db "commune/db/matrix/gen"
	"commune/gomatrix"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) DomainAPIEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := chi.URLParam(r, "domain")

		if strings.Contains(domain, "localhost") {
			domain = "http://" + domain
		} else if !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}

		domain = fmt.Sprintf("%s/.well-known/api", domain)

		resp, err := http.Get(domain)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"exists": false,
				},
			})
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"exists": false,
				},
			})
			return
		}

		type Response struct {
			URL      string `json:"url"`
			MediaURL string `json:"media_url"`
		}

		var response Response
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Println("Error:", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"exists": false,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"url":       response.URL,
				"media_url": response.MediaURL,
			},
		})
	}
}

type CreateSpaceRequest struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Topic     string `json:"topic"`
	AvatarURL string `json:"avatar_url"`
	Private   bool   `json:"private"`
}

func (c *App) CreateSpace() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &CreateSpaceRequest{})

		if err != nil {
			log.Println(err)
			RespondWithBadRequestError(w)
			return
		}

		user := c.LoggedInUser(r)

		if c.Config.Restrictions.Space.RequireVerification && !user.Admin {

			verified, err := c.MatrixDB.Queries.IsVerifed(context.Background(), user.MatrixUserID)
			if !verified || err != nil {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":     "You must verify your email to create a space.",
						"forbidden": true,
					},
				})
				return
			}
		}

		if c.Config.Restrictions.Space.SenderAge > 0 && !user.Admin {
			valid := c.IsSenderAgeValid(user, c.Config.Restrictions.Space.SenderAge)
			if !valid {

				day := "day"
				if c.Config.Restrictions.Space.SenderAge > 1 {
					day = "days"
				}

				msg := fmt.Sprintf("Your account needs to be at least %d %s old to create a space.", c.Config.Restrictions.Space.SenderAge, day)

				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":     msg,
						"forbidden": true,
					},
				})
				return
			}
		}

		spaces, err := c.MatrixDB.Queries.GetUserSpaces(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Could not create space.",
				},
			})
			return
		}

		if len(spaces) >= c.Config.Restrictions.Space.SpacesPerUser && !user.Admin {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":     "You have reached the maximum number of spaces.",
					"forbidden": true,
				},
			})
			return
		}

		if p.Private {
			p.Username = RandomString(8)
		}

		valid := IsValidAlias(p.Username)
		if !valid {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":     "That space name is not valid.",
					"forbidden": true,
				},
			})
			return
		}

		alias := c.ConstructMatrixRoomID(p.Username)

		if c.Config.Restrictions.Space.RejectReservedKeywords {

			reserved := IsKeywordReserved(p.Username)
			if reserved {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":  "That space name is not available.",
						"exists": reserved,
					},
				})
				return
			}
		}

		exists, err := c.MatrixDB.Queries.DoesSpaceExist(context.Background(), alias)
		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "There was an error creating your space.",
				},
			})
			return
		}

		if exists && !p.Private {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "That space name is not available.",
					"exists": exists,
				},
			})
			return
		}

		if exists && p.Private {
			p.Username = p.Username + RandomString(4)
		}

		_, err = c.NewSpace(&NewSpaceParams{
			Space:             p,
			MatrixUserID:      user.MatrixUserID,
			MatrixAccessToken: user.MatrixAccessToken,
		})

		if err != nil {
			log.Println("error creating space: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":   "Could not create space.",
					"message": err.Error(),
				},
			})
			return
		}

		details, err := c.MatrixDB.Queries.GetSpaceInfo(context.Background(), matrix_db.GetSpaceInfoParams{
			RoomAlias: pgtype.Text{
				String: strings.ToLower(alias),
				Valid:  true,
			},
			Creator: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
		})

		if err != nil {
			log.Println("error getting space info: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Space created but could not get details.",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success": true,
				"space":   details,
			},
		})

	}
}

type NewSpaceParams struct {
	Space             *CreateSpaceRequest
	MatrixUserID      string
	MatrixAccessToken string
}

func (c *App) NewSpace(p *NewSpaceParams) (string, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, p.MatrixUserID, p.MatrixAccessToken)
	if err != nil {
		log.Println(err)
		return "", err
	}

	pl := gomatrix.Event{
		Type: "m.room.power_levels",
		Content: map[string]interface{}{
			"ban": 60,
			"events": map[string]interface{}{
				"m.room.name":         60,
				"m.room.power_levels": 100,
				"m.room.create":       10,
				"m.space.child":       50,
				"m.space.parent":      50,
				"room.settings":       80,
			},
			"events_default": 10,
			"invite":         10,
			"kick":           60,
			"notifications": map[string]interface{}{
				"room": 20,
			},
			"redact":        10,
			"state_default": 10,
			"users": map[string]interface{}{
				p.MatrixUserID: 100,
				//c.DefaultMatrixAccount: 100,
			},
			"users_default": 10,
		},
	}

	hv := "world_readable"
	ga := "can_join"

	if p.Space.Private {
		hv = "shared"
		ga = "forbidden"
	}

	initState := []gomatrix.Event{
		gomatrix.Event{
			Type: "m.room.history_visibility",
			Content: map[string]interface{}{
				"history_visibility": hv,
			},
		}, gomatrix.Event{
			Type: "m.room.guest_access",
			Content: map[string]interface{}{
				"guest_access": ga,
			},
		}, gomatrix.Event{
			Type: "m.room.name",
			Content: map[string]interface{}{
				"name": p.Space.Name,
			},
		}, gomatrix.Event{
			Type: "m.room.topic",
			Content: map[string]interface{}{
				"topic": p.Space.Topic,
			},
		}, gomatrix.Event{
			Type: "m.restrict_events_to",
			Content: map[string]interface{}{
				"age":      0,
				"verified": true,
			},
		}, gomatrix.Event{
			Type: "m.space.type",
			Content: map[string]interface{}{
				"type": "board",
			},
		},
		pl,
	}

	if p.Space.AvatarURL != "" {
		initState = append(initState, gomatrix.Event{
			Type: "m.room.avatar",
			Content: map[string]interface{}{
				"url": p.Space.AvatarURL,
			},
		})
	}

	username := p.Space.Username

	creq := &gomatrix.ReqCreateRoom{
		RoomAliasName: username,
		Preset:        "public_chat",
		Visibility:    "public",
		CreationContent: map[string]interface{}{
			"type": "m.space",
		},
		InitialState: initState,
	}
	if p.Space.Private {
		creq.Preset = "private_chat"
		creq.Visibility = "private"
	}

	log.Println("creating actual room...")
	crr, err := matrix.CreateRoom(creq)

	if err != nil || crr == nil {
		log.Println(err)
		return "", err
	}

	log.Println("Was default space created?", crr)

	return crr.RoomID, nil

}

type CreateSpaceRoomRequest struct {
	Name   string `json:"name"`
	RoomID string `json:"room_id"`
}

func (c *App) CreateSpaceRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &CreateSpaceRoomRequest{})

		if err != nil {
			log.Println(err)
			RespondWithBadRequestError(w)
			return
		}

		user := c.LoggedInUser(r)

		room, err := c.NewSpaceRoom(&NewSpaceRoomParams{
			SpaceRoomID:       p.RoomID,
			Name:              p.Name,
			MatrixUserID:      user.MatrixUserID,
			MatrixAccessToken: user.MatrixAccessToken,
		})

		if err != nil || room == "" {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":   "room could not be created",
					"success": false,
				},
			})
			return
		}

		sse, err := c.NewStateEvent(&NewStateEventParams{
			RoomID:            p.RoomID,
			EventType:         "m.space.child",
			StateKey:          room,
			MatrixUserID:      user.MatrixUserID,
			MatrixAccessToken: user.MatrixAccessToken,
			Content: map[string]interface{}{
				"via":       []string{c.Config.Matrix.PublicServer},
				"suggested": false,
			},
		})

		if err != nil || sse == "" {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":   "room created but space relationship could not be created",
					"success": false,
				},
			})
			return
		}

		state, err := c.MatrixDB.Queries.GetRoomState(context.Background(), matrix_db.GetRoomStateParams{
			RoomID: pgtype.Text{
				String: room,
				Valid:  true,
			},
			UserID: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
		})
		if err != nil {
			log.Println(err)
		}

		if state != nil {
			log.Println("state: ", state)
		}

		var st RoomState
		err = json.Unmarshal(state, &st)
		if err != nil {
			log.Println("Error unmarshalling state: ", err)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success": true,
				"room_id": room,
				"state":   st,
			},
		})

	}
}

type NewSpaceRoomParams struct {
	SpaceRoomID       string
	Name              string
	MatrixUserID      string
	MatrixAccessToken string
}

func (c *App) NewSpaceRoom(p *NewSpaceRoomParams) (string, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, p.MatrixUserID, p.MatrixAccessToken)
	if err != nil {
		log.Println(err)
		return "", err
	}

	pl := gomatrix.Event{
		Type: "m.room.power_levels",
		Content: map[string]interface{}{
			"ban": 60,
			"events": map[string]interface{}{
				"m.room.name":         60,
				"m.room.power_levels": 100,
				"m.room.create":       10,
				"m.space.child":       50,
				"m.space.parent":      50,
				"room.settings":       80,
			},
			"events_default": 10,
			"invite":         10,
			"kick":           60,
			"notifications": map[string]interface{}{
				"room": 20,
			},
			"redact":        10,
			"state_default": 10,
			"users": map[string]interface{}{
				p.MatrixUserID: 100,
				//c.DefaultMatrixAccount: 100,
			},
			"users_default": 10,
		},
	}

	initState := []gomatrix.Event{
		gomatrix.Event{
			Type: "m.room.history_visibility",
			Content: map[string]interface{}{
				"history_visibility": "world_readable",
			},
		}, gomatrix.Event{
			Type: "m.room.guest_access",
			Content: map[string]interface{}{
				"guest_access": "can_join",
			},
		}, gomatrix.Event{
			Type: "m.room.name",
			Content: map[string]interface{}{
				"name": p.Name,
			},
		}, gomatrix.Event{
			Type: "m.space.child.type",
			Content: map[string]interface{}{
				"type": "board",
			},
		}, gomatrix.Event{
			Type:     "m.space.parent",
			StateKey: &p.SpaceRoomID,
			Content: map[string]interface{}{
				"via":       []string{c.Config.Matrix.PublicServer},
				"canonical": true,
			},
		}, gomatrix.Event{
			Type: "m.restrict_events_to",
			Content: map[string]interface{}{
				"age":      0,
				"verified": true,
			},
		}, gomatrix.Event{
			Type: "m.space.type",
			Content: map[string]interface{}{
				"type": "board",
			},
		},
		pl,
	}

	creq := &gomatrix.ReqCreateRoom{
		RoomAliasName: RandomString(30),
		Preset:        "public_chat",
		Visibility:    "public",
		InitialState:  initState,
	}

	crr, err := matrix.CreateRoom(creq)

	if err != nil || crr == nil {
		log.Println(err)
		return "", err
	}

	log.Println("Was space room created?", crr)

	return crr.RoomID, nil

}

type NewStateEventParams struct {
	Content           any
	RoomID            string
	EventType         string
	StateKey          string
	MatrixUserID      string
	MatrixAccessToken string
}

func (c *App) NewStateEvent(p *NewStateEventParams) (string, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, p.MatrixUserID, p.MatrixAccessToken)
	if err != nil {
		log.Println(err)
		return "", err
	}

	sse, err := matrix.SendStateEvent(p.RoomID, p.EventType, p.StateKey, p.Content)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return sse.EventID, nil

}

func (c *App) GetDefaultSpaces() (*[]matrix_db.GetDefaultSpacesRow, error) {
	spaces, err := c.MatrixDB.Queries.GetDefaultSpaces(context.Background(), pgtype.Text{
		String: c.Config.Matrix.PublicServer,
		Valid:  true,
	})
	if err != nil {
		return nil, err
	}
	return &spaces, nil
}

func (c *App) DefaultSpaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		spaces, err := c.GetDefaultSpaces()
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "error getting default spaces",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"spaces": spaces,
			},
		})

	}
}

func (c *App) GetAllSpaces() (*[]matrix_db.GetAllSpacesRow, error) {
	spaces, err := c.MatrixDB.Queries.GetAllSpaces(context.Background())
	if err != nil {
		return nil, err
	}
	return &spaces, nil
}

func (c *App) AllSpaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		spaces, err := c.GetAllSpaces()
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "error getting default spaces",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"spaces": spaces,
			},
		})

	}
}

func (c *App) GetSpaceEmoji() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		emoji, err := c.MatrixDB.Queries.GetUserSpacesEmoji(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"emoji": emoji,
			},
		})

	}
}
