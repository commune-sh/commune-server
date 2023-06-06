package app

import (
	"log"
	"net/http"
)

func (c *App) CreateSpace() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			Name      string `json:"name"`
			Alias     string `json:"alias"`
			Topic     string `json:"topic"`
			AvatarURL string `json:"avatar_url"`
			Private   bool   `json:"private"`
		}{})

		if err != nil {
			log.Println(err)
			RespondWithBadRequestError(w)
			return
		}

		user := c.LoggedInUser(r)

		log.Println("who is user?", user.Username)

		log.Println("payload is ", p)

		/*
			log.Println("what is room id ????", p.RoomID, p.Content)

			serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

			matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
			if err != nil {
				log.Println(err)
			}

			resp, err := matrix.SendMessageEvent(p.RoomID, "m.room.message", p.Content)
			if err != nil {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":   err,
						"success": "false",
					},
				})
				return
			}

			slug := resp.EventID[len(resp.EventID)-11:]

			item, err := c.MatrixDB.Queries.GetSpaceEvent(context.Background(), slug)

			if err != nil {
				log.Println("error getting event: ", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error": "event created but could not be fetched",
					},
				})
				return
			}

			json, err := gabs.ParseJSON([]byte(item.JSON.String))
			if err != nil {
				log.Println("error parsing json: ", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusInternalServerError,
					JSON: map[string]any{
						"error": "event not found",
					},
				})
				return
			}

			s := ProcessComplexEvent(&EventProcessor{
				EventID:     item.EventID,
				JSON:        json,
				Slug:        item.Slug,
				DisplayName: item.DisplayName.String,
				RoomAlias:   item.RoomAlias.String,
				AvatarURL:   item.AvatarUrl.String,
				ReplyCount:  item.Replies,
				Reactions:   item.Reactions,
			})

			if p.IsReply && p.InThread != "" {
				//slug := p.InThread[len(p.InThread)-11:]
				go c.UpdateEventRepliesCache(p.InThread, p.RoomID)
			} else {
				go c.UpdateSpaceEventsCache(p.RoomID)
			}
		*/

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success": "false",
			},
		})

	}
}
