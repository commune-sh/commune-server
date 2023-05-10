package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	matrix_db "shpong/db/matrix/gen"
	"shpong/gomatrix"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/unrolled/secure"
)

func (c *App) AllEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// get events for this space
		events, err := c.MatrixDB.Queries.GetEvents(context.Background(), pgtype.Int8{
			Int64: time.Now().UnixMilli(),
			Valid: true,
		})

		if err != nil {
			log.Println("error getting events: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "internal server error",
				},
			})
			return
		}

		var items []interface{}

		for _, item := range events {

			json, err := gabs.ParseJSON([]byte(item.JSON.String))
			if err != nil {
				log.Println("error parsing json: ", err)
			}

			s := ProcessComplexEvent(&EventProcessor{
				EventID:     item.EventID,
				Slug:        item.Slug,
				RoomAlias:   GetLocalPart(item.RoomAlias.String),
				JSON:        json,
				DisplayName: item.DisplayName.String,
				AvatarURL:   item.AvatarUrl.String,
				ReplyCount:  item.Replies,
				Reactions:   item.Reactions,
			})

			items = append(items, s)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"events": items,
			},
		})

	}
}

func (c *App) GetEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		event := chi.URLParam(r, "event")

		log.Println("event id is ", event)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"event": event,
			},
		})

	}
}

func (c *App) NewPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			RoomID  string `json:"room_id"`
			Content any    `json:"content"`
		}{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		user := c.LoggedInUser(r)

		log.Println("what is room id ????", p.RoomID, p.Content)

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			log.Println(err)
		}

		resp, err := matrix.SendMessageEvent(user.UserSpaceID, "m.room.message", p.Content)

		log.Println("resp is ", resp, err)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"event": "test",
			},
		})

	}
}

func (c *App) GetEventReplies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		event := chi.URLParam(r, "event")

		log.Println("event id is ", event)

		replies, err := c.MatrixDB.Queries.GetSpaceEventReplies(context.Background(), event)

		if err != nil {
			log.Println("error getting event replies: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "couldn't get event replies",
					"exists": false,
				},
			})
			return
		}

		var items []interface{}

		for _, item := range replies {

			json, err := gabs.ParseJSON([]byte(item.JSON.String))
			if err != nil {
				log.Println("error parsing json: ", err)
			}

			s := ProcessComplexEvent(&EventProcessor{
				EventID:     item.EventID,
				Slug:        item.Slug,
				JSON:        json,
				DisplayName: item.DisplayName.String,
				AvatarURL:   item.AvatarUrl.String,
				Reactions:   item.Reactions,
			})

			items = append(items, s)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"replies": items,
			},
		})

	}
}

func (c *App) SpaceEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		space := chi.URLParam(r, "space")

		alias := c.ConstructMatrixRoomID(space)

		// check if space exists in DB
		state, err := c.MatrixDB.Queries.GetSpaceState(context.Background(), alias)

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "space does not exist",
					"exists": false,
				},
			})
			return
		}

		// get all space room state events
		//state_events, err := c.MatrixDB.Queries.GetSpaceState(context.Background(), alias)

		sps := ProcessState(state)

		sreq := matrix_db.GetSpaceEventsParams{
			OriginServerTS: pgtype.Int8{
				Int64: time.Now().UnixMilli(),
				Valid: true,
			},
			RoomAlias: alias,
		}

		// get events for this space
		events, err := c.MatrixDB.Queries.GetSpaceEvents(context.Background(), sreq)

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "internal server error",
				},
			})
			return
		}

		var items []interface{}

		for _, item := range events {

			json, err := gabs.ParseJSON([]byte(item.JSON.String))
			if err != nil {
				log.Println("error parsing json: ", err)
			}

			s := ProcessComplexEvent(&EventProcessor{
				EventID:     item.EventID,
				Slug:        item.Slug,
				JSON:        json,
				RoomAlias:   space,
				DisplayName: item.DisplayName.String,
				AvatarURL:   item.AvatarUrl.String,
				ReplyCount:  item.Replies,
				Reactions:   item.Reactions,
			})

			items = append(items, s)
		}

		isMember := false

		user := c.LoggedInUser(r)

		if user != nil {

			mem, err := c.MatrixDB.Queries.IsUserSpaceMember(context.Background(), matrix_db.IsUserSpaceMemberParams{
				UserID: pgtype.Text{
					String: user.MatrixUserID,
					Valid:  true,
				},
				RoomID: pgtype.Text{
					String: state.RoomID,
					Valid:  true,
				},
			})
			if err != nil {
				log.Println("error getting event: ", err)
			}
			isMember = mem
		}

		log.Println("is member is ", isMember)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"state":  sps,
				"events": items,
			},
		})

	}
}

func (c *App) SpaceEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//user := c.LoggedInUser(r)

		space := chi.URLParam(r, "space")

		slug := chi.URLParam(r, "slug")

		alias := c.ConstructMatrixRoomID(space)

		sreq := matrix_db.GetSpaceEventParams{
			EventID:   slug,
			RoomAlias: alias,
		}

		item, err := c.MatrixDB.Queries.GetSpaceEvent(context.Background(), sreq)

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "event not found",
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
			Slug:        slug,
			JSON:        json,
			DisplayName: item.DisplayName.String,
			AvatarURL:   item.AvatarUrl.String,
			ReplyCount:  item.Replies,
			Reactions:   item.Reactions,
		})

		// get event replies
		eventReplies, err := c.MatrixDB.Queries.GetSpaceEventReplies(context.Background(), item.EventID)

		if err != nil {
			log.Println("error getting event replies: ", err)
		}

		var replies []interface{}
		{

			for _, item := range eventReplies {

				json, err := gabs.ParseJSON([]byte(item.JSON.String))
				if err != nil {
					log.Println("error parsing json: ", err)
				}

				s := ProcessComplexEvent(&EventProcessor{
					EventID:     item.EventID,
					Slug:        item.Slug,
					JSON:        json,
					DisplayName: item.DisplayName.String,
					AvatarURL:   item.AvatarUrl.String,
					Reactions:   item.Reactions,
				})

				replies = append(replies, s)
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"event":   s,
				"replies": replies,
			},
		})
	}
}

/*
func (c *App) UserEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		username := chi.URLParam(r, "username")
		log.Println("username is: ", username)

		sender := c.ConstructMatrixID(username)
		alias := c.ConstructMatrixUserRoomID(username)

		log.Println("sender is: ", sender, alias)

		events, err := c.MatrixDB.Queries.GetUserEvents(context.Background(), matrix_db.GetUserEventsParams{
			Sender: pgtype.Text{
				String: sender,
				Valid:  true,
			},
			RoomAlias: alias,
		})

		if err != nil {
			log.Println("error getting event: ", err)
		}

		var items []interface{}

		for _, item := range events {
			json, err := gabs.ParseJSON([]byte(item.JSON.String))
			if err != nil {
				log.Println("error parsing json: ", err)
			}

			s := ProcessEvent(item.EventID, item.Slug.String, json)

			items = append(items, s)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"events": items,
			},
		})

	}
}

func (c *App) UserEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		username := chi.URLParam(r, "username")

		slug := chi.URLParam(r, "slug")

		sender := c.ConstructMatrixID(username)
		alias := c.ConstructMatrixUserRoomID(username)

		event, err := c.MatrixDB.Queries.GetEvent(context.Background(), matrix_db.GetEventParams{
			Sender: pgtype.Text{
				String: sender,
				Valid:  true,
			},
			Slug: pgtype.Text{
				String: slug,
				Valid:  true,
			},
			RoomAlias: alias,
		})

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "event not found",
				},
			})
			return
		}

		json, err := gabs.ParseJSON([]byte(event.JSON.String))
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

		s := ProcessEvent(event.EventID, slug, json)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"event": s,
			},
		})
	}
}
*/

func (c *App) UserPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		username := chi.URLParam(r, "username")

		eventID := chi.URLParam(r, "eventID")

		log.Println("username is: ", username, eventID)

		us := c.LoggedInUser(r)
		type NotFoundPage struct {
			LoggedInUser interface{}
			AppName      string
			Nonce        string
			Secret       string
		}

		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
		claims["iat"] = time.Now().Unix()
		claims["name"] = "lol whut"
		claims["email"] = "test@test.com"

		key := []byte(c.Config.App.JWTKey)
		tokenString, err := token.SignedString(key)
		if err != nil {
			log.Println(err)
		}

		t, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return key, nil
		})

		if c, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
			log.Println(c["name"], c["email"])
		} else {
			log.Println(err)
		}

		nonce := secure.CSPNonce(r.Context())
		pg := NotFoundPage{
			LoggedInUser: us,
			AppName:      c.Config.Name,
			Secret:       tokenString,
			Nonce:        nonce,
		}

		c.Templates.ExecuteTemplate(w, "index-user", pg)
	}
}
