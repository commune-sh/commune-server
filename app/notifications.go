package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	matrix_db "shpong/db/matrix/gen"
	"sync"

	"github.com/gorilla/websocket"
)

type Notification struct {
	MatrixUserID string `json:"matrix_user_id"`
	JSON         []byte `json:"json"`
}

type NotificationClient struct {
	Conn  *websocket.Conn
	Token string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var connectedClients = make(map[string]*websocket.Conn)
var clientsMutex sync.Mutex

func (c *App) SyncNotifications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		token := query.Get("token")

		log.Println("token is ", token)

		user, err := c.GetTokenUser(token)
		if err != nil || user == nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": false,
				},
			})
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade connection to WebSocket:", err)
			return
		}
		defer conn.Close()

		clientsMutex.Lock()
		connectedClients[user.MatrixUserID] = conn
		clientsMutex.Unlock()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read message:", err)
				break
			}

			err = conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("Failed to write message:", err)
				break
			}
		}

		clientsMutex.Lock()
		delete(connectedClients, token)
		clientsMutex.Unlock()

	}
}

func (c *App) sendNotification(mid string, json []byte) {
	clientsMutex.Lock()
	conn, found := connectedClients[mid]
	clientsMutex.Unlock()

	if found {
		err := conn.WriteMessage(websocket.TextMessage, json)
		if err != nil {
			log.Println("Failed to send notification to client:", err)
			conn.Close()
			clientsMutex.Lock()
			delete(connectedClients, mid)
			clientsMutex.Unlock()
		}
	} else {
		log.Println("No client with the provided token:", mid)
	}
}

func (c *App) GetNotifications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		items, err := c.MatrixDB.Queries.GetUserNotifications(context.Background(), user.MatrixUserID)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error":   "Could not get notifications",
					"success": false,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"notifications": items,
			},
		})

	}
}

func (c *App) MarkRead() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		err := c.MatrixDB.Queries.MarkAsRead(context.Background(), user.MatrixUserID)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error":   "Could not get notifications",
					"success": false,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success": true,
			},
		})

	}
}

type NotificationParams struct {
	ThreadEventID  string
	ReplyToEventID string
	User           *User
	ReplyEvent     *Event
}

func (c *App) NewReplyNotification(n *NotificationParams) error {

	eventID := n.ReplyToEventID

	slug := eventID[len(eventID)-11:]

	// get event
	replyingToEvent, err := c.GetEvent(&GetEventParams{
		Slug: slug,
	})

	if err != nil || replyingToEvent == nil {
		log.Println("error getting thread event", err)
		return err
	}

	// don't create notification if replying/reacting to self
	replyingToSelf := replyingToEvent.Sender.ID == n.User.MatrixUserID
	log.Println("is replying to self?", replyingToSelf)
	if replyingToSelf {
		return nil
	}

	notificationType := "post.reply"

	if replyingToEvent.EventID != n.ThreadEventID {
		notificationType = "reply.reply"
	}

	np := matrix_db.CreateNotificationParams{
		FromMatrixUserID: n.User.MatrixUserID,
		ForMatrixUserID:  replyingToEvent.Sender.ID,
		RelatesToEventID: replyingToEvent.EventID,
		EventID:          n.ReplyEvent.EventID,
		ThreadEventID:    n.ThreadEventID,
		Type:             notificationType,
		Body:             "",
		RoomAlias:        replyingToEvent.RoomAlias,
		RoomID:           replyingToEvent.RoomID,
	}

	js, ok := n.ReplyEvent.Content.(map[string]interface{})
	if ok {
		body, ok := js["body"].(string)
		if ok {
			x := body
			if len(x) > 100 {
				x = x[:100]
			}
			np.Body = x
		}
	}

	notification, err := c.MatrixDB.Queries.CreateNotification(context.Background(), np)

	if err != nil {
		log.Println("notification could not be created")
		return err
	}

	serialized, err := json.Marshal(notification)
	if err != nil {
		log.Println(err)
	}

	c.sendNotification(replyingToEvent.Sender.ID, serialized)

	return nil
}

func (c *App) NewReactionNotification(n *NotificationParams) error {

	eventID := n.ReplyToEventID

	slug := eventID[len(eventID)-11:]

	// get event
	replyingToEvent, err := c.GetEvent(&GetEventParams{
		Slug: slug,
	})

	if err != nil || replyingToEvent == nil {
		log.Println("error getting thread event", err)
		return err
	}

	// don't create notification if replying/reacting to self
	replyingToSelf := replyingToEvent.Sender.ID == n.User.MatrixUserID
	log.Println("is replying to self?", replyingToSelf)
	if replyingToSelf {
		return nil
	}

	notificationType := "reaction"

	np := matrix_db.CreateNotificationParams{
		FromMatrixUserID: n.User.MatrixUserID,
		ForMatrixUserID:  replyingToEvent.Sender.ID,
		RelatesToEventID: replyingToEvent.EventID,
		EventID:          n.ReplyEvent.EventID,
		Type:             notificationType,
		Body:             "",
		RoomAlias:        replyingToEvent.RoomAlias,
		RoomID:           replyingToEvent.RoomID,
	}

	js, ok := n.ReplyEvent.Content.(map[string]interface{})
	if ok {

		log.Println("reaction event content", js)
		rt, ok := js["m.relates_to"].(map[string]interface{})

		if ok {
			key, ok := rt["key"].(string)
			if ok {
				np.Body = key
			}
		}

	}

	notification, err := c.MatrixDB.Queries.CreateNotification(context.Background(), np)

	if err != nil {
		log.Println("notification could not be created")
		return err
	}

	serialized, err := json.Marshal(notification)
	if err != nil {
		log.Println(err)
	}

	c.sendNotification(replyingToEvent.Sender.ID, serialized)

	return nil
}

type JoinNotificationParams struct {
	User   *User
	Space  string
	RoomID string
}

func (c *App) NewJoinNotification(n *JoinNotificationParams) error {

	mid := fmt.Sprintf("%s:%s", n.Space, c.Config.Matrix.PublicServer)

	np := matrix_db.CreateNotificationParams{
		FromMatrixUserID: n.User.MatrixUserID,
		ForMatrixUserID:  mid,
		Type:             "space.follow",
		Body:             "",
		RoomAlias:        n.Space,
		RoomID:           n.RoomID,
	}

	notification, err := c.MatrixDB.Queries.CreateNotification(context.Background(), np)

	if err != nil {
		log.Println("notification could not be created")
		return err
	}

	serialized, err := json.Marshal(notification)
	if err != nil {
		log.Println(err)
	}

	c.sendNotification(mid, serialized)

	return nil
}
