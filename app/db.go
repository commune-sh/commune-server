package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"shpong/config"
	matrix_db "shpong/db/matrix/gen"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/lib/pq"
)

type DB struct {
	*pgxpool.Pool
}

type MatrixDB struct {
	*pgxpool.Pool
	Queries *matrix_db.Queries
}

// NewDB returns a new database instace
func NewMatrixDB() (*MatrixDB, error) {

	c, err := config.Read(CONFIG_FILE)
	if err != nil {
		panic(err)
	}

	address := c.DB.Matrix

	conn, err := pgxpool.New(context.Background(), address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	err = conn.Ping(context.Background())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	q := matrix_db.New(conn)

	store := &MatrixDB{conn, q}

	return store, nil
}

func (c *App) StartNotifyListener() {

	c.MatrixDB.Exec(context.Background(), "LISTEN events_notification")
	n, err := c.MatrixDB.Pool.Acquire(context.Background())
	if err != nil {
		log.Println("error acquiring pool: ", err)
	}
	m := n.Hijack()

	for {
		x, err := m.WaitForNotification(context.Background())
		if err != nil {
			log.Println("error acquiring pool: ", err)
		}

		if x != nil && x.Payload != "" {

			log.Println("PAYLOAD IS", x)

			type NotifyEvent struct {
				EventID       string `json:"event_id"`
				RoomID        string `json:"room_id"`
				Type          string `json:"type"`
				TransactionID string `json:"txn_id"`
			}

			ne := NotifyEvent{}

			err := json.Unmarshal([]byte(x.Payload), &ne)

			if err != nil {
				log.Println("error unmarshalling payload: ", err)
			}

			if ne.Type == "m.room.member" {
				ms, err := c.MatrixDB.Queries.GetMembershipState(context.Background(), pgtype.Text{String: ne.EventID, Valid: true})
				if err != nil {
					log.Println(err)
				}

				if ms.Membership.String == "join" &&
					ms.SpaceAlias.String[0] == '@' &&
					ms.UserID.String != ms.Creator.String {

					n := Notification{
						FromMatrixUserID: ms.UserID.String,
						DisplayName:      ms.DisplayName.String,
						AvatarURL:        ms.AvatarUrl.String,
						CreatedAt:        ms.OriginServerTS.Int64,
						Type:             "space.follow",
					}

					serialized, err := json.Marshal(n)
					if err != nil {
						log.Println(err)
					}

					c.sendNotification(ms.Creator.String, serialized)
				}
			}

			eventID := ne.EventID

			slug := eventID[len(eventID)-11:]

			event, err := c.GetEvent(&GetEventParams{
				Slug: slug,
			})

			log.Println("GOT NOTIFIED with new event", event)

			if err == nil && event.Type == "m.room.message" ||
				event.Type == "m.room.member" ||
				event.Type == "m.room.name" ||
				event.Type == "m.room.topic" ||
				event.Type == "m.reaction" ||
				event.Type == "space.board.post" ||
				event.Type == "m.room.redaction" {

				serialized, err := json.Marshal(event)
				if err != nil {
					log.Println(err)
				} else {
					c.sendMessageNotification(event.RoomID, serialized)
				}
				continue
			}

			if err == nil {
				n, err := c.MatrixDB.Queries.GetNotification(context.Background(), eventID)
				if err != nil {
					log.Println(err)
				} else {

					if n.ForMatrixUserID.String == event.Sender.ID {
						continue
					}

					serialized, err := json.Marshal(n)
					if err != nil {
						log.Println(err)
					}

					c.sendNotification(n.ForMatrixUserID.String, serialized)
				}
				continue
			}
		}
	}

}
