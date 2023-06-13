package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shpong/gomatrix"
	"time"

	matrix_db "shpong/db/matrix/gen"

	"github.com/Jeffail/gabs/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

type NewPostBody struct {
	RoomID   string `json:"room_id"`
	Content  any    `json:"content"`
	IsReply  bool   `json:"is_reply"`
	InThread string `json:"in_thread"`
	Type     string `json:"type"`
}

type NewPostParams struct {
	Body              *NewPostBody
	MatrixUserID      string
	MatrixAccessToken string
}

func (c *App) NewPost(p *NewPostParams) (*Event, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, p.MatrixUserID, p.MatrixAccessToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	resp, err := matrix.SendMessageEvent(p.Body.RoomID, p.Body.Type, p.Body.Content)
	if err != nil {
		return nil, err
	}

	slug := resp.EventID[len(resp.EventID)-11:]

	item, err := c.MatrixDB.Queries.GetSpaceEvent(context.Background(), slug)

	if err != nil {
		log.Println("error getting event: ", err)
		return nil, err
	}

	json, err := gabs.ParseJSON([]byte(item.JSON.String))
	if err != nil {
		log.Println("error parsing json: ", err)
		return nil, err
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

	if p.Body.IsReply && p.Body.InThread != "" {
		go c.UpdateEventRepliesCache(p.Body.InThread, p.Body.RoomID)
	} else {
		go c.UpdateSpaceEventsCache(p.Body.RoomID)
	}

	go c.UpdateIndexEventsCache()

	return &s, nil
}

func (c *App) CreatePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &NewPostBody{})

		if err != nil {
			log.Println(err)
			RespondWithBadRequestError(w)
			return
		}

		user := c.LoggedInUser(r)

		event, err := c.NewPost(&NewPostParams{
			Body:              p,
			MatrixUserID:      user.MatrixUserID,
			MatrixAccessToken: user.MatrixAccessToken,
		})

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success": "true",
				"event":   event,
			},
		})

	}
}

func (c *App) UpdateSpaceEventsCache(roomID string) error {

	log.Println("updating cache for space", roomID)

	sreq := matrix_db.GetSpaceEventsParams{
		OriginServerTS: pgtype.Int8{
			Int64: time.Now().UnixMilli(),
			Valid: true,
		},
		RoomID: roomID,
	}

	events, err := c.MatrixDB.Queries.GetSpaceEvents(context.Background(), sreq)

	if err != nil {
		log.Println("error getting event: ", err)
		return err
	}

	var items []interface{}

	for _, item := range events {

		json, err := gabs.ParseJSON([]byte(item.JSON.String))
		if err != nil {
			log.Println("error parsing json: ", err)
			return err
		}

		s := ProcessComplexEvent(&EventProcessor{
			EventID:     item.EventID,
			Slug:        item.Slug,
			JSON:        json,
			RoomAlias:   item.RoomAlias.String,
			DisplayName: item.DisplayName.String,
			AvatarURL:   item.AvatarUrl.String,
			ReplyCount:  item.Replies,
			Reactions:   item.Reactions,
		})

		items = append(items, s)
	}

	serialized, err := json.Marshal(items)
	if err != nil {
		log.Println(err)
		return err
	}

	err = c.Cache.Events.Set(roomID, serialized, 0).Err()
	if err != nil {
		log.Println(err)
		return err
	}

	go c.UpdateIndexEventsCache()

	return nil
}

func (c *App) UpdateIndexEventsCache() error {

	log.Println("updating cache for index")

	ge := pgtype.Int8{
		Int64: time.Now().UnixMilli(),
		Valid: true,
	}

	events, err := c.MatrixDB.Queries.GetEvents(context.Background(), ge)

	if err != nil {
		log.Println("error getting events: ", err)
		return err
	}

	var items []interface{}

	for _, item := range events {

		json, err := gabs.ParseJSON([]byte(item.JSON.String))
		if err != nil {
			log.Println("error parsing json: ", err)
			return err
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

	serialized, err := json.Marshal(items)
	if err != nil {
		log.Println(err)
		return err
	}

	err = c.Cache.Events.Set("index", serialized, 0).Err()
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (c *App) UpdateEventRepliesCache(event string, roomID string) error {
	log.Println("updating cache for event slug", event)
	replies, err := c.MatrixDB.Queries.GetSpaceEventReplies(context.Background(), event)

	if err != nil {
		log.Println("error getting event replies: ", err)
		return err
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
			RoomAlias:   item.RoomAlias.String,
			AvatarURL:   item.AvatarUrl.String,
			Reactions:   item.Reactions,
		})

		items = append(items, s)
	}

	serialized, err := json.Marshal(items)
	if err != nil {
		log.Println(err)
		return err
	}

	err = c.Cache.Events.Set(event, serialized, 0).Err()
	if err != nil {
		log.Println(err)
		return err
	}

	go c.UpdateSpaceEventsCache(roomID)
	go c.UpdateIndexEventsCache()

	return nil
}
