package app

import (
	"context"
	"log"
	"net/http"
	db "shpong/db/gen"
)

func (c *App) GetNotifications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		items, err := c.DB.Queries.GetUserNotifications(context.Background(), user.MatrixUserID)
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

		err := c.DB.Queries.MarkAsRead(context.Background(), user.MatrixUserID)
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

	np := db.CreateNotificationParams{
		FromMatrixUserID: n.User.MatrixUserID,
		ForMatrixUserID:  replyingToEvent.Sender.ID,
		RelatesToEventID: replyingToEvent.EventID,
		EventID:          n.ReplyEvent.EventID,
		ThreadEventID:    n.ThreadEventID,
		Type:             notificationType,
		Body:             "",
		RoomAlias:        replyingToEvent.RoomAlias,
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

	_, err = c.DB.Queries.CreateNotification(context.Background(), np)

	if err != nil {
		log.Println("notification could not be created")
		return err
	}

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

	np := db.CreateNotificationParams{
		FromMatrixUserID: n.User.MatrixUserID,
		ForMatrixUserID:  replyingToEvent.Sender.ID,
		RelatesToEventID: replyingToEvent.EventID,
		EventID:          n.ReplyEvent.EventID,
		Type:             notificationType,
		Body:             "",
		RoomAlias:        replyingToEvent.RoomAlias,
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

	_, err = c.DB.Queries.CreateNotification(context.Background(), np)

	if err != nil {
		log.Println("notification could not be created")
		return err
	}

	return nil
}
