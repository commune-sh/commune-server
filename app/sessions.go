package app

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
)

type User struct {
	Username          string `json:"username"`
	DisplayName       string `json:"display_name"`
	AvatarURL         string `json:"avatar_url"`
	AccessToken       string `json:"access_token"`
	MatrixAccessToken string `json:"matrix_access_token"`
	MatrixUserID      string `json:"matrix_user_id"`
	MatrixDeviceID    string `json:"matrix_device_id"`
	UserID            string `json:"user_id"`
	UserSpaceID       string `json:"user_space_id"`
	Email             string `json:"email"`
	Age               int64  `json:"age"`
	Admin             bool   `json:"admin"`
}

func NewSession(sec string) *sessions.CookieStore {
	s := sessions.NewCookieStore([]byte(sec))
	s.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 365,
		HttpOnly: false,
	}
	return s
}

func GetSession(r *http.Request, c *App) (*sessions.Session, error) {
	s, err := c.Sessions.Get(r, c.Config.App.CookieName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return s, nil
}

func (c *App) StoreUserSession(u *User) error {
	log.Println("creating session for user: ", u)

	serialized, err := json.Marshal(u)
	if err != nil {
		log.Println(err)
		return err
	}

	err = c.SessionsStore.Set(u.AccessToken, serialized, 0).Err()
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (c *App) GetTokenUser(token string) (*User, error) {

	user, err := c.SessionsStore.Get(token).Result()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var us User
	err = json.Unmarshal([]byte(user), &us)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &us, nil

}
