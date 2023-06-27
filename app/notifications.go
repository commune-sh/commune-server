package app

import "log"

type NotificationParams struct {
	ThreadEventID  string
	ReplyToEventID string
	User           *User
}

func (c *App) NewNotification(n *NotificationParams) error {

	nestedReply := n.ReplyToEventID != n.ThreadEventID

	log.Println("building new notification", n)

	if !nestedReply {
		eventID := n.ThreadEventID

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

		replyingToSelf := event.Sender.ID == n.User.MatrixUserID
		log.Println("is replying to self?", replyingToSelf)
		if replyingToSelf {
			return nil
		}
	}

	return nil
}
