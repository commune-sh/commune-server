package app

import (
	"context"
	"encoding/json"
	"log"
	db "shpong/db/gen"
)

type NotificationParams struct {
	ThreadEventID  string
	ReplyToEventID string
	User           *User
}

func (c *App) NewNotification(n *NotificationParams) error {

	nestedReply := n.ReplyToEventID != n.ThreadEventID

	log.Println("building new notification", n)

	eventID := n.ReplyToEventID

	slug := eventID[len(eventID)-11:]
	event, err := c.GetEvent(&GetEventParams{
		Slug: slug,
	})
	if err == nil && event != nil {
		log.Println("got thread event", event)
	} else {
		log.Println("error getting thread event", err)
		return err
	}

	if !nestedReply {
		replyingToSelf := event.Sender.ID == n.User.MatrixUserID
		log.Println("is replying to self?", replyingToSelf)
		if replyingToSelf {
			return nil
		}

		np := db.CreateNotificationParams{
			MatrixUserID: n.User.MatrixUserID,
			Type:         "reply",
		}

		js, err := json.Marshal(event.Content)
		if err != nil {
			log.Println(err)
			return err
		}

		if js != nil {
			np.Content = js
		}

		_, err = c.DB.Queries.CreateNotification(context.Background(), np)

		if err != nil {
			log.Println("notification could not be created")
			return err
		}
	}

	return nil
}
