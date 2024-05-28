package app

import (
	matrix_db "commune/db/matrix/gen"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) CreateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		p, err := ReadRequestJSON(r, w, &struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
			Session  string `json:"session"`
			Code     string `json:"code"`
		}{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		isValid := false

		if c.Config.Features.RequireEmail {
			valid, err := c.DoesEmailCodeExist(&CodeVerification{
				Email:   p.Email,
				Code:    p.Code,
				Session: p.Session,
			})

			if err != nil || !valid {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"valid": valid,
					},
				})
				return
			}
			isValid = valid

		}

		p.Username = strings.ToLower(p.Username)

		if !c.Config.Features.RegistrationEnabled {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"created": false,
					"error":   "registration is disabled",
				},
			})
			return
		}

		/*
				if c.Config.Auth.BlockPopularEmailProviders {

					// let's ban the most common email providers to prevent spam
					// from /static/emails.json

					banned := IsEmailBanned(p.Email)
					if banned {
						RespondWithJSON(w, &JSONResponse{
							Code: http.StatusOK,
							JSON: map[string]any{
								"created":            false,
								"provider_forbidden": true,
								"error":              "email provider is not allowed",
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
								"error":   "email provider does not exist",
							},
						})
						return
					}
				}

				// check to see if username already exists
				/*
					exists, _ := c.MatrixDB.Queries.DoesUsernameExist(context.Background(), pgtype.Text{
				String: username,
				Valid:  true,
			})

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
		*/

		// check to see if matrix account already exists

		valid := IsValidAlias(p.Username)
		if !valid {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":     "That username is not valid.",
					"forbidden": true,
				},
			})
			return
		}

		mname := fmt.Sprintf(`@%s:%s`, p.Username, c.Config.Matrix.Homeserver)
		exists, err := c.MatrixDB.Queries.DoesMatrixUserExist(context.Background(), pgtype.Text{String: mname, Valid: true})

		if exists || err != nil {
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
					"error":   "That username is invalid. Try another username.",
				},
			})
			return
		}

		if c.Config.Features.RequireEmail && isValid {
			err = c.MatrixDB.Queries.VerifyEmail(context.Background(), matrix_db.VerifyEmailParams{
				Email: pgtype.Text{
					String: p.Email,
					Valid:  true,
				},
				MatrixUserID: pgtype.Text{
					String: resp.Response.UserID,
					Valid:  true,
				},
			})
			if err != nil {
				log.Println(err)

				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error": "could not create account",
					},
				})
				return
			}
		}

		token := RandomString(32)

		cr := time.Now().Unix()

		created, err := c.MatrixDB.Queries.GetUserCreatedAt(context.Background(), pgtype.Text{String: resp.Response.UserID, Valid: true})
		if err != nil {
			log.Println(err)
		} else {
			cr = created.Int64
		}

		user := &User{
			//UserID:   idu,
			Username:          p.Username,
			DisplayName:       p.Username,
			Email:             p.Email,
			AccessToken:       token,
			MatrixAccessToken: resp.Response.AccessToken,
			MatrixUserID:      resp.Response.UserID,
			MatrixDeviceID:    resp.Response.DeviceID,
			UserSpaceID:       resp.UserSpaceID,
			Age:               cr,
		}

		pubkey, err := c.CreateNewUserKey(resp.Response.UserID)

		if err == nil && pubkey != nil {
			user.PublicKeyPem = *pubkey
		}

		if c.Config.Features.RequireEmail && isValid {
			user.Verified = true
		}

		err = c.StoreUserSession(user)

		spaces, err := c.MatrixDB.Queries.GetUserSpaces(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}
		rooms, err := c.MatrixDB.Queries.GetJoinedRooms(context.Background(), pgtype.Text{String: user.MatrixUserID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		// send success JSON
		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"created":      true,
				"access_token": token,
				"credentials":  user,
				"spaces":       spaces,
				"rooms":        rooms,
			},
		})
	}
}

func (c *App) UsernameAvailable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		username := chi.URLParam(r, "username")

		exists, err := c.MatrixDB.Queries.DoesUsernameExist(context.Background(), pgtype.Text{
			String: username,
			Valid:  true,
		})
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

		if exists {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"available": false,
				},
			})
			return
		}

		mname := fmt.Sprintf(`@%s:%s`, username, c.Config.Matrix.Homeserver)
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

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"available": !exists,
			},
		})
	}
}

func (c *App) ValidateEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		email := chi.URLParam(r, "email")

		if c.Config.Authentication.BlockPopularEmailProviders {

			banned := IsEmailBanned(email)
			if banned {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"available":          false,
						"provider_forbidden": true,
						"error":              "email provider is not allowed",
					},
				})
				return
			}
		}

		if c.Config.Authentication.QueryMXRecords {

			provider := strings.Split(email, "@")[1]
			records, err := net.LookupMX(provider)
			if err != nil || len(records) == 0 {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"available": false,
						"error":     "email provider does not exist",
					},
				})
				return
			}
		}

		exists, err := c.MatrixDB.Queries.DoesEmailExist(context.Background(), pgtype.Text{
			String: email,
			Valid:  true,
		})

		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "could not check if email exists",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"available": !exists,
			},
		})
	}
}
