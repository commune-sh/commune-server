package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	matrix_db "shpong/db/matrix/gen"
	"shpong/gomatrix"
	"strconv"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/unrolled/secure"
)

func (c *App) AllEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		if user != nil {
			log.Println("user is ", user.Username)
		}

		query := r.URL.Query()
		last := query.Get("last")

		// get events for this space

		ge := pgtype.Int8{
			Int64: time.Now().UnixMilli(),
			Valid: true,
		}

		if last != "" {
			i, _ := strconv.ParseInt(last, 10, 64)
			log.Println(i)
			ge.Int64 = i
		}

		if c.Config.Cache.IndexEvents {

			// get events for this space from cache
			cached, err := c.Cache.Events.Get("index").Result()
			if err != nil {
				log.Println("index events not in cache")
			}

			if cached != "" {
				var events []Event
				err = json.Unmarshal([]byte(cached), &events)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("responding with cached events")

					RespondWithJSON(w, &JSONResponse{
						Code: http.StatusOK,
						JSON: map[string]any{
							"events": events,
						},
					})
					return
				}
			}
		}

		events, err := c.MatrixDB.Queries.GetEvents(context.Background(), ge)

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
				RoomAlias:   item.RoomAlias.String,
				JSON:        json,
				DisplayName: item.DisplayName.String,
				AvatarURL:   item.AvatarUrl.String,
				ReplyCount:  item.Replies,
				Reactions:   item.Reactions,
			})

			items = append(items, s)
		}

		go func() {
			if c.Config.Cache.IndexEvents {

				serialized, err := json.Marshal(items)
				if err != nil {
					log.Println(err)
				}

				err = c.Cache.Events.Set("index", serialized, 0).Err()
				if err != nil {
					log.Println(err)
				}
			}

		}()

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"events": items,
			},
		})

	}
}

func (c *App) UserFeedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		query := r.URL.Query()
		last := query.Get("last")

		// get events for this space

		ge := pgtype.Int8{
			Int64: time.Now().UnixMilli(),
			Valid: true,
		}

		if last != "" {
			i, _ := strconv.ParseInt(last, 10, 64)
			log.Println(i)
			ge.Int64 = i
		}

		fe := matrix_db.GetUserFeedEventsParams{
			OriginServerTS: ge,
			UserID: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
		}

		events, err := c.MatrixDB.Queries.GetUserFeedEvents(context.Background(), fe)

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

		//user := c.LoggedInUser(r)

		//space := chi.URLParam(r, "space")

		//alias := c.ConstructMatrixRoomID(space)

		item, err := c.MatrixDB.Queries.GetSpaceEvent(context.Background(), event)

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "event not found",
					"exists": false,
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

func (c *App) SSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		token := query.Get("token")

		user, err := c.GetTokenUser(token)
		if err != nil || token == "" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("starting to sync for", user.Username)

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			log.Println(err)
		}

		//events := []any{}

		// Create a new channel to send events to the client
		eventCh := make(chan any)

		syncer := matrix.Syncer.(*gomatrix.DefaultSyncer)
		syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
			//fmt.Println("Message: ", ev)
			//events = append(events, ev)
			eventCh <- ev
		})

		go func() {
			for {
				if err := matrix.Sync(); err != nil {
					fmt.Println("Sync() returned ", err)
				}
				// Optional: Wait a period of time before trying to sync again.
			}
		}()

		log.Println("sending SSE to ", user.Username)

		// Set the content type to text/event-stream
		w.Header().Set("Content-Type", "text/event-stream")
		// Set cache-control header to prevent caching of the response
		w.Header().Set("Cache-Control", "no-cache")
		// Set connection header to keep the connection open
		w.Header().Set("Connection", "keep-alive")

		// Continuously listen for events and write them to the response

		for {
			event := <-eventCh
			data, err := json.Marshal(event)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher, ok := w.(http.Flusher)
			if ok {
				flusher.Flush()
			}
		}

	}
}

func generateEvent() string {
	// Generate a random event
	return time.Now().Format("2006-01-02 15:04:05")
}

func (c *App) Sync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		log.Println("starting to sync for", user.Username)

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			log.Println(err)
		}

		events := []any{}

		syncer := matrix.Syncer.(*gomatrix.DefaultSyncer)
		syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
			fmt.Println("Message: ", ev)
			events = append(events, ev)
		})

		query := r.URL.Query()
		timeout := query.Get("timeout")

		go func() {
			for {
				if err := matrix.Sync(); err != nil {
					fmt.Println("Sync() returned ", err)
				}
				// Optional: Wait a period of time before trying to sync again.
			}
		}()

		if timeout != "" {
			log.Println("got since", timeout)
			duration, err := time.ParseDuration(timeout + "ms")
			if err != nil {
				fmt.Println("Error parsing duration:", err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusInternalServerError,
					JSON: map[string]any{
						"error": "bad since value",
					},
				})
				return
			}

			time.Sleep(duration)
		}
		matrix.StopSync()

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"syncing": true,
				"events":  events,
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

		user := c.LoggedInUser(r)

		space := chi.URLParam(r, "space")

		alias := c.ConstructMatrixRoomID(space)

		ssp := matrix_db.GetSpaceStateParams{
			RoomAlias: alias,
		}

		if user != nil && user.MatrixUserID != "" {
			ssp.UserID = pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			}
		}

		// check if space exists in DB
		state, err := c.MatrixDB.Queries.GetSpaceState(context.Background(), ssp)

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

		hideRoom := state.IsPublic.Bool != state.Joined
		log.Println("should we hide room? ", hideRoom)

		// get all space room state events
		//state_events, err := c.MatrixDB.Queries.GetSpaceState(context.Background(), alias)

		sps := ProcessState(state)

		sreq := matrix_db.GetSpaceEventsParams{
			OriginServerTS: pgtype.Int8{
				Int64: time.Now().UnixMilli(),
				Valid: true,
			},
			RoomID: state.RoomID,
		}

		query := r.URL.Query()
		last := query.Get("last")

		// get events for this space

		if last != "" {
			i, _ := strconv.ParseInt(last, 10, 64)
			log.Println(i)
			sreq.OriginServerTS.Int64 = i
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

		/*
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
				if mem {
					sps.Joined = true
				}
			}
		*/

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"state":  sps,
				"events": items,
			},
		})

	}
}

func (c *App) SpaceRoomEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		space := chi.URLParam(r, "space")
		room := chi.URLParam(r, "room")

		log.Println("space is", space)
		log.Println("room is", room)

		alias := c.ConstructMatrixRoomID(space)

		ssp := matrix_db.GetSpaceStateParams{
			RoomAlias: alias,
		}

		if user != nil && user.MatrixUserID != "" {
			ssp.UserID = pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			}
		}

		// check if space exists in DB
		state, err := c.MatrixDB.Queries.GetSpaceState(context.Background(), ssp)

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

		sps := ProcessState(state)

		scp := matrix_db.GetSpaceChildParams{
			ParentRoomAlias: pgtype.Text{
				String: alias,
				Valid:  true,
			},
			ChildRoomAlias: pgtype.Text{
				String: room,
				Valid:  true,
			},
		}

		if user != nil {
			log.Println("user is ", user.MatrixUserID)
			scp.UserID = pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			}
		}

		crs, err := c.MatrixDB.Queries.GetSpaceChild(context.Background(), scp)

		if err != nil || crs.ChildRoomID.String == "" {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "space room does not exist",
					"state":  sps,
					"exists": false,
				},
			})
			return
		}
		log.Println("what is child room ID?", crs.ChildRoomID)

		sreq := matrix_db.GetSpaceEventsParams{
			OriginServerTS: pgtype.Int8{
				Int64: time.Now().UnixMilli(),
				Valid: true,
			},
			RoomID: crs.ChildRoomID.String,
		}

		query := r.URL.Query()
		last := query.Get("last")

		// get events for this space

		if last != "" {
			i, _ := strconv.ParseInt(last, 10, 64)
			log.Println(i)
			sreq.OriginServerTS.Int64 = i
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
			if mem {
				sps.Joined = true
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"state":      sps,
				"room_state": crs,
				"events":     items,
			},
		})

	}
}

func (c *App) SpaceEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//user := c.LoggedInUser(r)

		//space := chi.URLParam(r, "space")

		slug := chi.URLParam(r, "slug")

		//alias := c.ConstructMatrixRoomID(space)

		item, err := c.MatrixDB.Queries.GetSpaceEvent(context.Background(), slug)

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "event not found",
					"exists": false,
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
