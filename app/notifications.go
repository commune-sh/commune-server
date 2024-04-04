package app

import (
	"context"
	"log"
	"net/http"
	matrix_db "commune/db/matrix/gen"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
)

type Notification struct {
	Type             string `json:"type"`
	FromMatrixUserID string `json:"from_matrix_user_id"`
	EventID          string `json:"event_id"`
	ThreadEventID    string `json:"thread_event_id"`
	CreatedAt        int64  `json:"created_at"`
	Body             string `json:"body"`
	DisplayName      string `json:"display_name"`
	AvatarURL        string `json:"avatar_url"`
	Read             bool   `json:"read"`
	RoomAlias        string `json:"room_alias"`
	RelatesToEventID string `json:"relates_to_event_id"`
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

var connectedClients = make(map[string][]*websocket.Conn)
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
		if _, ok := connectedClients[user.MatrixUserID]; ok {
			log.Println("already exists in list, adding new client")
			connectedClients[user.MatrixUserID] = append(connectedClients[user.MatrixUserID], conn)
		} else {
			log.Println("adding new client")
			connectedClients[user.MatrixUserID] = []*websocket.Conn{conn}
		}
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
		delete(connectedClients, user.MatrixUserID)
		clientsMutex.Unlock()

	}
}

func (c *App) sendNotification(mid string, json []byte) {
	clientsMutex.Lock()
	conns, found := connectedClients[mid]
	clientsMutex.Unlock()

	if found {
		for _, conn := range conns {
			err := conn.WriteMessage(websocket.TextMessage, json)
			if err != nil {
				log.Println("Failed to send notification to client:", err)
				conn.Close()
				clientsMutex.Lock()
				delete(connectedClients, mid)
				clientsMutex.Unlock()
			}
		}
	} else {
		log.Println("No client with the provided token:", mid)
	}
}

func (c *App) GetNotifications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		last, err := c.Cache.Notifications.Get(user.MatrixUserID).Result()
		if err != nil {
			log.Println(err)
		}

		gn := matrix_db.GetNotificationsParams{
			Sender: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			RoomID: pgtype.Text{
				String: user.UserSpaceID,
				Valid:  true,
			},
			OriginServerTS: pgtype.Int8{
				Int64: time.Now().UnixMilli(),
				Valid: true,
			},
		}

		if last != "" {
			i, _ := strconv.ParseInt(last, 10, 64)
			log.Println(i)
			gn.OriginServerTS.Int64 = i
		}

		items, err := c.MatrixDB.Queries.GetNotifications(context.Background(), gn)
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

		query := r.URL.Query()
		last := query.Get("last")

		log.Println(last)

		user := c.LoggedInUser(r)

		err := c.Cache.Notifications.Set(user.MatrixUserID, last, 0).Err()
		if err != nil {
			log.Println(err)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success": true,
			},
		})

	}
}
