package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

func (c *App) Middleware() {

	if c.Config.Mode == "development" {
		c.Router.Use(c.reloadtemplates)
	}
}

func (c *App) LoggedInUser(r *http.Request) *User {
	token, ok := r.Context().Value("token").(string)

	if !ok {
		return nil
	}

	user, err := c.GetTokenUser(token)
	if err != nil {
		log.Println(err)
		return nil
	}

	return user
}

// Checks for logged in user on routes that use it
func (c *App) GetAuthorizationToken(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		at, err := ExtractAccessToken(r)
		if err != nil {
			log.Println(err)
			h.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), "token", at.Token)

		h.ServeHTTP(w, r.WithContext(ctx))

	})
}

// makes sure this route is autehnticated
func (c *App) RequireAuthentication(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		s, err := GetSession(r, c)
		if err != nil {
			log.Println(err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		token, ok := s.Values["access_token"].(string)

		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userid, err := c.SessionsStore.Get(token).Result()
		if err != nil {
			log.Println(err)
			h.ServeHTTP(w, r)
			return
		}

		user, err := c.SessionsStore.Get(userid).Result()
		if err != nil {
			log.Println(err)
			h.ServeHTTP(w, r)
			return
		}

		var us User
		err = json.Unmarshal([]byte(user), &us)
		if err != nil || us.UserID == "" {
			log.Println(err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// makes sure this route is autehnticated
func (c *App) GuestsOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		s, err := GetSession(r, c)
		if err != nil {
			log.Println(err)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		token, ok := s.Values["access_token"].(string)

		if ok && len(token) > 0 {
			userid, err := c.SessionsStore.Get(token).Result()
			if err == nil && userid != "" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		h.ServeHTTP(w, r)
	})
}

func (c *App) reloadtemplates(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.ReloadTemplates()

		h.ServeHTTP(w, r)
	})
}
