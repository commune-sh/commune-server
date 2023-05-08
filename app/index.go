package app

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/unrolled/secure"
)

func (c *App) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		us := c.LoggedInUser(r)
		type NotFoundPage struct {
			LoggedInUser interface{}
			AppName      string
			Nonce        string
			Secret       string
		}

		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
		claims["iat"] = time.Now().Unix()
		claims["name"] = "lol whut"
		claims["email"] = "test@test.com"

		key := []byte(c.Config.App.JWTKey)
		tokenString, err := token.SignedString(key)
		if err != nil {
			log.Println(err)
		}

		t, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return key, nil
		})

		if c, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
			log.Println(c["name"], c["email"])
		} else {
			log.Println(err)
		}

		nonce := secure.CSPNonce(r.Context())
		pg := NotFoundPage{
			LoggedInUser: us,
			AppName:      c.Config.Name,
			Secret:       tokenString,
			Nonce:        nonce,
		}

		c.Templates.ExecuteTemplate(w, "index", pg)
	}
}
