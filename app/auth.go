package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	matrix_db "shpong/db/matrix/gen"

	"shpong/gomatrix"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tidwall/buntdb"
)

func (c *App) ValidateLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := ReadRequestJSON(r, w, &struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		log.Println("recieved payload ", p)

		creds, err := c.DB.Queries.GetCredentials(context.Background(), p.Username)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetCredentials failed: %v\n", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"authenticated": false,
					"exists":        false,
					"error":         "username or email does not exist",
				},
			})
			return
		}

		serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

		matrix, err := gomatrix.NewClient(serverName, "", "")
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"authenticated": false,
					"error":         "internal server error",
				},
			})
			return
		}

		rl := &gomatrix.ReqLogin{
			Type:     "m.login.password",
			User:     p.Username,
			Password: p.Password,
		}

		resp, err := matrix.Login(rl)
		if err != nil || resp == nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"authenticated": false,
					"error":         "username or password is incorrect",
				},
			})
			return
		}

		if resp != nil {
			log.Println("resp is ", resp)
		}

		profile, err := c.MatrixDB.Queries.GetProfile(context.Background(), p.Username)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CreateUser failed: %v\n", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"authenticated": false,
					"error":         "internal server error",
				},
			})
			return
		}

		token := RandomString(32)

		idu := encodeUUID(creds.ID.Bytes)

		room_alias := fmt.Sprintf("#@%s:%s", p.Username, c.Config.Matrix.PublicServer)
		creator := fmt.Sprintf("@%s:%s", p.Username, c.Config.Matrix.PublicServer)

		userspace, err := c.MatrixDB.Queries.GetUserSpaceID(context.Background(), matrix_db.GetUserSpaceIDParams{
			RoomAlias: room_alias,
			Creator: pgtype.Text{
				String: creator,
				Valid:  true,
			},
		})
		if err != nil {
			log.Println(err)
		}

		admin, err := c.MatrixDB.Queries.IsAdmin(context.Background(), pgtype.Text{String: resp.UserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		user := &User{
			UserID:            idu,
			Username:          p.Username,
			Email:             creds.Email,
			DisplayName:       profile.Displayname.String,
			AvatarURL:         profile.AvatarUrl.String,
			AccessToken:       token,
			MatrixAccessToken: resp.AccessToken,
			MatrixUserID:      resp.UserID,
			MatrixDeviceID:    resp.DeviceID,
			UserSpaceID:       userspace,
			Age:               creds.CreatedAt.Time.Unix(),
			Verified:          creds.Verified,
			Admin:             admin,
		}

		err = c.StoreUserSession(user)

		spaces, err := c.MatrixDB.Queries.GetUserSpaces(context.Background(), pgtype.Text{String: resp.UserID, Valid: true})
		if err != nil {
			log.Println(err)
		}
		rooms, err := c.MatrixDB.Queries.GetJoinedRooms(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"authenticated": true,
				"access_token":  token,
				"credentials":   user,
				"spaces":        spaces,
				"rooms":         rooms,
			},
		})

	}
}

func (c *App) ValidateSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		at, err := ExtractAccessToken(r)

		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusUnauthorized,
				JSON: map[string]any{
					"valid": false,
				},
			})
			return
		}

		user, err := c.GetTokenUser(at.Token)
		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": false,
				},
			})
			return
		}

		exists, err := c.MatrixDB.Queries.IsAccessTokenValid(context.Background(), matrix_db.IsAccessTokenValidParams{
			UserID: user.MatrixUserID,
			Token:  user.MatrixAccessToken,
		})
		if err != nil || !exists {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusUnauthorized,
				JSON: map[string]any{
					"valid": false,
				},
			})
			return
		}

		spaces, err := c.MatrixDB.Queries.GetUserSpaces(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		rooms, err := c.MatrixDB.Queries.GetJoinedRooms(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		admin, err := c.MatrixDB.Queries.IsAdmin(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}
		user.Admin = admin

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"valid":       true,
				"credentials": user,
				"spaces":      spaces,
				"rooms":       rooms,
			},
		})

	}
}

func (c *App) ValidateToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		at, err := ExtractAccessToken(r)

		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusUnauthorized,
				JSON: map[string]any{
					"valid": false,
				},
			})
			return
		}

		user, err := c.GetTokenUser(at.Token)
		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": false,
				},
			})
			return
		}

		exists, err := c.MatrixDB.Queries.IsAccessTokenValid(context.Background(), matrix_db.IsAccessTokenValidParams{
			UserID: user.MatrixUserID,
			Token:  user.MatrixAccessToken,
		})
		if err != nil || !exists {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusUnauthorized,
				JSON: map[string]any{
					"valid": false,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"valid": true,
			},
		})

	}
}

func (c *App) SendCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type request struct {
			Email   string `json:"email"`
			Session string `json:"session"`
		}

		p, err := ReadRequestJSON(r, w, &request{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		log.Println("recieved payload ", p)

		// check to see if this email domain is allowed
		/*
			banned := IsEmailBanned(p.Email)
			if banned {
				log.Println("This email is forbidden.")
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error": "Email is banned",
					},
				})
				return
			}
		*/

		code := GenerateMagicCode()

		log.Println("magic code is ", code, p.Session)

		//
		//go c.SendSignupCode(p.Email, code)
		//

		err = c.AddCodeToCache(p.Email, &CodeVerification{
			Code:    code,
			Session: p.Session,
			Email:   p.Email,
		})

		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "code could not be sent",
					"sent":  false,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"sent": true,
			},
		})
	}
}

type CodeVerification struct {
	Email   string `json:"email"`
	Session string `json:"session"`
	Code    string `json:"code"`
}

func (c *App) VerifyCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &CodeVerification{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		valid, err := c.DoesEmailCodeExist(p)

		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": valid,
					"error": "code could not be verified",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"valid": valid,
			},
		})
	}
}

func (c *App) OldVerifyEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type request struct {
			Email string `json:"email"`
		}

		p, err := ReadRequestJSON(r, w, &request{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		log.Println("recieved payload ", p)

		banned := IsEmailBanned(p.Email)
		if banned {
			log.Println("email is banned")
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Email is banned",
				},
			})
			return
		}

		//err := c.SendSignupVerificationEmail(p.Email)

		err = c.Cache.VerificationCodes.View(func(tx *buntdb.Tx) error {
			val, err := tx.Get("mykey")
			if err != nil {
				return err
			}
			fmt.Printf("value is %s\n", val)
			return nil
		})

		type response struct {
			Sent  bool   `json:"sent"`
			Token string `json:"token"`
		}

		at, err := ExtractAccessToken(r)

		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusUnauthorized,
				JSON: map[string]any{
					"error": "unauthorized",
				},
			})

			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"token": at.Token,
			},
		})
	}
}
