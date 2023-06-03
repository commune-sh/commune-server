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
		type NotFoundPage struct {
			LoggedInUser interface{}
			AppName      string
			Nonce        string
			Secret       string
		}

		nonce := secure.CSPNonce(r.Context())
		pg := NotFoundPage{
			LoggedInUser: us,
			AppName:      c.Config.Name,
			Nonce:        nonce,
		}

		c.Templates.ExecuteTemplate(w, "index", pg)
	}
}
