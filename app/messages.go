package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	matrix_db "shpong/db/matrix/gen"
	"strconv"
	"sync"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) RoomMessages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		room := chi.URLParam(r, "room")

		log.Println("room is", room)

		// get events for this space
		events, err := c.GetSpaceMessages(&SpaceMessagesParams{
			RoomID: room,
			Last:   r.URL.Query().Get("last"),
			After:  r.URL.Query().Get("after"),
			Topic:  r.URL.Query().Get("topic"),
		})

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		// get events for this space

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"events": events,
			},
		})

	}
}

type SpaceMessagesParams struct {
	RoomID string
	Last   string
	After  string
	Topic  string
}

func (c *App) GetSpaceMessages(p *SpaceMessagesParams) (*[]Event, error) {

	sreq := matrix_db.GetSpaceMessagesParams{
		RoomID: p.RoomID,
	}

	if len(p.Topic) > 0 {
		sreq.Topic = pgtype.Text{
			String: p.Topic,
			Valid:  true,
		}
	}

	// get events for this space

	if p.Last != "" && p.After == "" {
		i, _ := strconv.ParseInt(p.Last, 10, 64)
		log.Println("adding last", i)
		sreq.OriginServerTS = pgtype.Int8{
			Int64: i,
			Valid: true,
		}
	}

	if p.After != "" {
		i, _ := strconv.ParseInt(p.After, 10, 64)
		sreq.After = pgtype.Int8{
			Int64: i,
			Valid: true,
		}
	}

	// get events for this space
	events, err := c.MatrixDB.Queries.GetSpaceMessages(context.Background(), sreq)

	if err != nil {
		log.Println("error getting event: ", err)
		return nil, err
	}

	items := []Event{}

	for _, item := range events {

		json, err := gabs.ParseJSON([]byte(item.JSON.String))
		if err != nil {
			log.Println("error parsing json: ", err)
		}

		s := ProcessComplexEvent(&EventProcessor{
			EventID:          item.EventID,
			Slug:             item.Slug,
			JSON:             json,
			DisplayName:      item.DisplayName.String,
			AvatarURL:        item.AvatarUrl.String,
			ReplyCount:       item.Replies,
			Reactions:        item.Reactions,
			Edited:           item.Edited,
			EditedOn:         item.EditedOn,
			PrevContent:      item.PrevContent,
			Redacted:         item.Redacted,
			LastThreadReply:  item.LastThreadReply,
			ThreadReplyCount: item.ThreadReplies.Int64,
		})

		items = append(items, s)
	}

	return &items, nil
}

var messageClients = make(map[string][]*Client)
var messageClientsMutex sync.Mutex

type Client struct {
	Conn   *websocket.Conn
	RoomID string
}

func (c *App) SyncMessages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		roomID := chi.URLParam(r, "room")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade connection to WebSocket:", err)
			return
		}
		defer conn.Close()

		client := &Client{Conn: conn, RoomID: roomID}

		messageClientsMutex.Lock()
		messageClients[roomID] = append(messageClients[roomID], client)
		messageClientsMutex.Unlock()

		for {

			last := 0

			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read message:", err)
				break
			}

			type syncMessage struct {
				Type  string `json:"type"`
				Last  int    `json:"last"`
				Value string `json:"value"`
			}

			var sm syncMessage
			err = json.Unmarshal(msg, &sm)
			if err != nil {
				log.Println(err)
			}

			// let's check for messages since last sync
			if sm.Type == "sync" && sm.Last != last {
				events, err := c.GetSpaceMessages(&SpaceMessagesParams{
					RoomID: roomID,
					After:  strconv.Itoa(sm.Last),
				})

				if err != nil {
					log.Println("error getting event: ", err)
				}

				if events != nil {
					serialized, err := json.Marshal(events)
					if err != nil {
						log.Println(err)
					}
					err = conn.WriteMessage(websocket.TextMessage, serialized)
				}
			}

			last = sm.Last

			if sm.Type == "typing" && sm.Value != "" {
				c.sendMessageNotification(roomID, msg)
			}

		}

		messageClientsMutex.Lock()
		clients := messageClients[roomID]
		for i, c := range clients {
			if c == client {
				// Remove the client from the slice
				clients[i] = clients[len(clients)-1]
				clients = clients[:len(clients)-1]
				break
			}
		}
		messageClients[roomID] = clients
		messageClientsMutex.Unlock()

	}
}

func (c *App) sendMessageNotification(mid string, json []byte) {

	messageClientsMutex.Lock()
	clients := messageClients[mid]
	for _, client := range clients {
		err := client.Conn.WriteMessage(websocket.TextMessage, json)
		if err != nil {
			log.Println("Failed to send notification to client:", err)
			client.Conn.Close()
		}
	}
	messageClientsMutex.Unlock()

}

type GetEventThreadParams struct {
	EventID string
}

func (c *App) GetEventThread(p *GetEventThreadParams) (*[]*Event, error) {

	replies, err := c.MatrixDB.Queries.GetEventThread(context.Background(), p.EventID)

	if err != nil {
		log.Println("error getting event replies: ", err)
		return nil, err
	}

	var items []*Event

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
			Edited:      item.Edited,
			EditedOn:    item.EditedOn,
		})

		items = append(items, &s)
	}

	return &items, nil
}

func (c *App) EventThread() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		event := chi.URLParam(r, "event")

		slug := event[len(event)-11:]

		item, err := c.GetEvent(&GetEventParams{
			Slug: event,
		})

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "thread not found",
					"exists": false,
				},
			})
			return
		}

		replies, err := c.GetEventThread(&GetEventThreadParams{
			EventID: slug,
		})

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

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"event":  item,
				"events": replies,
			},
		})

	}
}
