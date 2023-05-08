package app

import (
	"context"
	db "shpong/db/gen"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) CreateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
		}{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		if c.Config.Auth.BlockPopularEmailProviders {

			// let's ban the most common email providers to prevent spam
			// from /static/emails.json

			banned := IsEmailBanned(p.Email)
			if banned {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"created": false,
						"error":   "email provider is not allowed",
					},
				})
				return
			}
		}

		if c.Config.Auth.QueryMXRecords {

			// let's look up MX records for the email domain
			// if there are no MX records, then we can't send an email
			// so we should reject the account creation

			provider := strings.Split(p.Email, "@")[1]
			records, err := net.LookupMX(provider)
			if err != nil || len(records) == 0 {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"created": false,
						"error":   "email does not look valid",
					},
				})
				return
			}
		}

		// check to see if username already exists
		exists, _ := c.DB.Queries.DoesUsernameExist(context.Background(), p.Username)

		if exists {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"created": false,
					"error":   "username already exists",
				},
			})
			return
		}

		// check to see if matrix account already exists

		mname := fmt.Sprintf(`@%s:%s`, p.Username, c.Config.Matrix.Homeserver)
		exists, _ = c.MatrixDB.Queries.DoesMatrixUserExist(context.Background(), pgtype.Text{String: mname, Valid: true})

		if exists {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"created": false,
					"error":   "matrix account already exists",
				},
			})
			return
		}

		// create the matrix account first
		resp, err := c.CreateMatrixUserAccount(p.Username, p.Password)

		log.Println("matrix account is: ", resp)

		if err != nil {

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"created": false,
					"error":   "could not create account",
				},
			})
			return
		}

		// hash the password
		hash, _ := HashPassword(p.Password)

		// create user
		id, err := c.DB.Queries.CreateUser(context.Background(), db.CreateUserParams{
			Email:    p.Email,
			Username: p.Username,
			Password: hash,
		})

		// send error JSON
		if err != nil {
			fmt.Fprintf(os.Stderr, "CreateUser failed: %v\n", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"created": false,
					"error":   "could not create account",
				},
			})
			return
		}

		idu := encodeUUID(id.Bytes)

		token := RandomString(32)

		err = c.StoreUserSession(&User{
			UserID:            idu,
			Username:          p.Username,
			Email:             p.Email,
			AccessToken:       token,
			MatrixAccessToken: resp.AccessToken,
			MatrixUserID:      resp.UserID,
			MatrixDeviceID:    resp.DeviceID,
		})

		// send success JSON
		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"created": true,
				"credentials": map[string]string{
					"id":                  idu,
					"access_token":        token,
					"matrix_user_id":      resp.UserID,
					"matrix_device_id":    resp.DeviceID,
					"matrix_access_token": resp.AccessToken,
					"display_name":        p.Username,
					"email":               p.Email,
				},
			},
		})
	}
}

/*
func (c *App) CreateMatrixAccount(username, hash string) error {

	_, err := c.MatrixDB.Queries.CreateUser(context.Background(), matrix_db.CreateUserParams{
		Name: pgtype.Text{
			String: fmt.Sprintf("@%s:%s", username, c.Config.Matrix.Homeserver),
			Valid:  true,
		},
		PasswordHash: pgtype.Text{
			String: hash,
			Valid:  true,
		},
		CreationTs: pgtype.Int8{
			Int64: time.Now().Unix(),
			Valid: true,
		},
		ShadowBanned: pgtype.Bool{
			Bool:  false,
			Valid: true,
		},
		Approved: pgtype.Bool{
			Bool:  true,
			Valid: true,
		},
	})

	if err != nil {
		return err
	}

	_, err = c.MatrixDB.Queries.CreateProfile(context.Background(), matrix_db.CreateProfileParams{
		UserID: username,
		Displayname: pgtype.Text{
			String: username,
			Valid:  true,
		},
	})
	if err != nil {
		log.Println(err)
	}

	_, err = c.MatrixDB.Queries.CreateUserDirectory(context.Background(), matrix_db.CreateUserDirectoryParams{
		UserID: fmt.Sprintf("@%s:%s", username, c.Config.Matrix.Homeserver),
		DisplayName: pgtype.Text{
			String: username,
			Valid:  true,
		},
	})

	if err != nil {
		return nil
	}

	at, _ := GenerateAccessToken()

	_, err = c.MatrixDB.Queries.CreateAccessToken(context.Background(), matrix_db.CreateAccessTokenParams{
		UserID: fmt.Sprintf("@%s:%s", username, c.Config.Matrix.Homeserver),
		DeviceID: pgtype.Text{
			String: RandomString(10),
			Valid:  true,
		},
		Token: at,
	})

	if err != nil {
		return nil
	}

	return nil
}
*/

func (c *App) UsernameAvailable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			Username string `json:"username"`
		}{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		exists, err := c.DB.Queries.DoesUsernameExist(context.Background(), p.Username)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DoesUsernameExist failed: %v\n", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "could not check if username exists",
				},
			})
			return
		}

		mname := fmt.Sprintf(`@%s:%s`, p.Username, c.Config.Matrix.Homeserver)
		exists, _ = c.MatrixDB.Queries.DoesMatrixUserExist(context.Background(), pgtype.Text{String: mname, Valid: true})

		if exists {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"created": false,
					"error":   "matrix account already exists",
				},
			})
			return
		}

		type Response struct {
			Exists bool `json:"exists"`
		}

		ff := Response{Exists: exists}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: ff,
		})
	}
}
