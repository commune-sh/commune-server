package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	matrix_db "shpong/db/matrix/gen"

	"shpong/gomatrix"

	"github.com/jackc/pgx/v5/pgtype"
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

		creds, err := c.MatrixDB.Queries.GetCredentials(context.Background(), pgtype.Text{
			String: strings.ToLower(p.Username),
			Valid:  true,
		})
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

		username := strings.ToLower(p.Username)

		if &creds.Email.String != nil {
			username = creds.Username
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
			User:     username,
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

		profile, err := c.MatrixDB.Queries.GetProfile(context.Background(), username)
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

		//idu := encodeUUID(creds.ID.Bytes)

		room_alias := fmt.Sprintf("#@%s:%s", username, c.Config.Matrix.PublicServer)
		creator := fmt.Sprintf("@%s:%s", username, c.Config.Matrix.PublicServer)

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
			//UserID:            idu,
			Username:          username,
			Email:             creds.Email.String,
			DisplayName:       profile.Displayname.String,
			AvatarURL:         profile.AvatarUrl.String,
			AccessToken:       token,
			MatrixAccessToken: resp.AccessToken,
			MatrixUserID:      resp.UserID,
			MatrixDeviceID:    resp.DeviceID,
			UserSpaceID:       userspace,
			Age:               creds.CreatedAt.Int64,
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
		/*

			pl, err := c.MatrixDB.Queries.GetUserPowerLevels(context.Background(), pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			})
			if err != nil {
				log.Panicln(err)
			}
		*/

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"authenticated": true,
				"access_token":  token,
				"credentials":   user,
				"spaces":        spaces,
				"rooms":         rooms,
				//"power_levels": pl,
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

		/*
			profile, err := c.MatrixDB.Queries.GetProfile(context.Background(), user.Username)
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
			user.DisplayName = profile.Displayname.String
			user.AvatarURL = profile.AvatarUrl.String
		*/

		spaces, err := c.MatrixDB.Queries.GetUserSpaces(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		admin, err := c.MatrixDB.Queries.IsAdmin(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}
		user.Admin = admin

		rooms, err := c.MatrixDB.Queries.GetJoinedRooms(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		/*
			pl, err := c.MatrixDB.Queries.GetUserPowerLevels(context.Background(), pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			})
			if err != nil {
				log.Panicln(err)
			}
		*/

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"valid":       true,
				"credentials": user,
				"spaces":      spaces,
				"rooms":       rooms,
				//"power_levels": pl,
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

		//check if email exists
		exists, err := c.MatrixDB.Queries.DoesEmailExist(context.Background(), pgtype.Text{
			String: p.Email,
			Valid:  true,
		})
		if err != nil {
			log.Println(err)
		}
		log.Println("does email exist?", exists)

		if exists {
			//don't send code to existing emails, silently ignore
			log.Println("ignore email")
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"sent": true,
				},
			})
			return
		}

		if c.Config.Authentication.BlockPopularEmailProviders {
			banned := IsEmailBanned(p.Email)
			if banned {
				log.Println("This email is forbidden.")
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"provider_forbidden": true,
						"error":              "email provider is not allowed",
					},
				})
				return
			}
		}

		code := GenerateMagicCode()

		log.Println("magic code is ", code, p)

		//
		go c.SendVerificationCode(p.Email, code)
		//

		err = c.AddCodeToCache(p.Session, &CodeVerification{
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
	Email    string `json:"email"`
	Session  string `json:"session"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (c *App) VerifyCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &CodeVerification{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		valid, err := c.DoesEmailCodeExist(p)

		if err != nil || !valid {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": valid,
				},
			})
			return
		}

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

		user.Email = p.Email
		user.Verified = true

		err = c.MatrixDB.Queries.VerifyEmail(context.Background(), matrix_db.VerifyEmailParams{
			Email: pgtype.Text{
				String: p.Email,
				Valid:  true,
			},
			MatrixUserID: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
		})
		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "could not update email",
				},
			})
			return
		}

		err = c.StoreUserSession(user)
		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "could not store user session",
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

func (c *App) VerifyEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &CodeVerification{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		valid, err := c.DoesEmailCodeExist(p)

		if err != nil || !valid {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": valid,
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

func (c *App) SendRecoveryCode() http.HandlerFunc {
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

		exists, err := c.MatrixDB.Queries.DoesEmailExist(context.Background(), pgtype.Text{
			String: p.Email,
			Valid:  true,
		})
		if err != nil {
			log.Println(err)
		}
		log.Println("does email exist?", exists)
		log.Println("does email exist?", exists)
		log.Println("does email exist?", exists)

		if exists {

			code := GenerateMagicCode()

			log.Println("magic code is ", code, p)

			//
			go c.SendVerificationCode(p.Email, code)
			//

			err = c.AddCodeToCache(p.Session, &CodeVerification{
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

		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"sent": true,
			},
		})
	}
}

func (c *App) VerifyRecoveryCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &CodeVerification{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		valid, err := c.DoesEmailCodeExist(p)

		if err != nil || !valid {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": valid,
				},
			})
			return
		}

		if valid {

			code := GenerateMagicCode()

			log.Println("new magic code is ", code, p)

			err = c.AddCodeToCache(p.Session, &CodeVerification{
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
					"valid": valid,
					"code":  code,
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

func (c *App) ResetPassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &CodeVerification{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		valid, err := c.DoesEmailCodeExist(p)

		if err != nil || !valid {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"valid": valid,
				},
			})
			return
		}

		if valid && p.Password != "" {

			hash, _ := HashPassword(p.Password)

			creds, err := c.MatrixDB.Queries.GetCredentials(context.Background(), pgtype.Text{
				String: p.Email,
				Valid:  true,
			})
			if err != nil || &creds == nil || creds.MatrixUserID.String == "" {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"success": false,
					},
				})
				return
			}

			err = c.MatrixDB.Queries.UpdatePassword(context.Background(), matrix_db.UpdatePasswordParams{
				PasswordHash: pgtype.Text{
					String: hash,
					Valid:  true,
				},
				Name: pgtype.Text{
					String: creds.MatrixUserID.String,
					Valid:  true,
				},
			})

			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"success": false,
					},
				})
				return
			}

		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"success": true,
			},
		})
	}
}

type PasswordUpdate struct {
	Password string `json:"password"`
	New      string `json:"new"`
}

func (c *App) UpdatePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &PasswordUpdate{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		user := c.LoggedInUser(r)

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
			User:     user.Username,
			Password: p.Password,
		}

		resp, err := matrix.Login(rl)
		if err != nil || resp == nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"success": false,
					"error":   "password is incorrect",
				},
			})
			return
		}

		hash, _ := HashPassword(p.New)

		err = c.MatrixDB.Queries.UpdatePassword(context.Background(), matrix_db.UpdatePasswordParams{
			PasswordHash: pgtype.Text{
				String: hash,
				Valid:  true,
			},
			Name: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
		})

		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
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

func (c *App) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		at, err := ExtractAccessToken(r)

		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusUnauthorized,
				JSON: map[string]any{
					"success": false,
				},
			})
			return
		}

		err = c.PurgeSession(at.Token)
		if err != nil {
			log.Println(err)

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusUnauthorized,
				JSON: map[string]any{
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
