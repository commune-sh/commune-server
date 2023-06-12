package app

import (
	"net/http"

	"github.com/unrolled/secure"
)

func (c *App) SSRIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))

	}
}

func (c *App) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		us := c.LoggedInUser(r)

		type Page struct {
			LoggedInUser interface{}
			AppName      string
			Nonce        string
			Secret       string
			Events       *[]Event
		}

		query := r.URL.Query()
		last := query.Get("last")

		// get events for this space

		events, err := c.GetIndexEvents(&IndexEventsParams{
			Last: last,
		})

		if err != nil {
			c.Error(w, r)
			return
		}

		nonce := secure.CSPNonce(r.Context())
		pg := Page{
			LoggedInUser: us,
			AppName:      c.Config.Name,
			Nonce:        nonce,
			Events:       events,
		}

		c.Templates.ExecuteTemplate(w, "index", pg)
	}
}
