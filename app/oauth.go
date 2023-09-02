package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	matrix_db "shpong/db/matrix/gen"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Banner        string `json:"banner"`
}

type AccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func (c *App) ValidateOauthDiscord() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		code := query.Get("code")

		log.Println("recieved code ", code)

		if code == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "missing provider or code",
				},
			})
			return
		}

		// Retrieve the required parameters from the request body.
		clientID := ""
		clientSecret := ""

		for _, item := range c.Config.Oauth {
			if item.Provider == "discord" && item.Enabled {
				clientID = item.ClientID
				clientSecret = item.ClientSecret
			}
		}

		if clientID == "" || clientSecret == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "this provider is not enabled",
				},
			})
			return
		}

		discordAPIEndpoint := "https://discord.com/api/v10/oauth2/token"

		redirectURI := fmt.Sprintf("%s/oauth/discord", c.Config.App.PublicDomain)

		data := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=authorization_code&code=%s&redirect_uri=%s", clientID, clientSecret, code, redirectURI)

		log.Println("url is ", data)

		resp, err := http.Post(discordAPIEndpoint, "application/x-www-form-urlencoded", strings.NewReader(data))
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Println(err, resp.StatusCode)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		log.Println("body is", resp.Body)

		var token AccessTokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		if token.AccessToken == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "missing access token",
				},
			})
			return
		}

		identityEndpoint := "https://discord.com/api/v10/users/@me"
		req, err := http.NewRequest("GET", identityEndpoint, nil)
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		req.Header.Set("Authorization", "Bearer "+token.AccessToken)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		var user DiscordUser
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		c.OauthUserSession(w, r, &OauthUser{
			ID:       user.ID,
			Username: user.GlobalName,
			Provider: "oidc-discord",
		})

	}
}

type OauthUser struct {
	ID       string
	Username string
	Provider string
}

func (c *App) OauthUserSession(w http.ResponseWriter, r *http.Request, u *OauthUser) {

	// Check if external user id exists
	userID, err := c.MatrixDB.Queries.GetExternalUserID(context.Background(), matrix_db.GetExternalUserIDParams{
		AuthProvider: u.Provider,
		ExternalID:   u.ID,
	})

	// if not create it
	if err != nil || userID == "" {
		log.Println(err)

		mid := fmt.Sprintf(`@%s:%s`, u.Username, c.Config.Matrix.PublicServer)
		exists, err := c.MatrixDB.Queries.DoesMatrixUserExist(context.Background(), pgtype.Text{String: mid, Valid: true})

		log.Println("does user id %s exist?", mid, exists)
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error checking if user exists",
				},
			})
			return
		}

		// if user id exists, append random string to username
		if exists {
			username := u.Username + RandomString(4)
			mid = fmt.Sprintf(`@%s:%s`, username, c.Config.Matrix.PublicServer)
		}

		log.Println("we'll create user")

		muser, err := c.MatrixDB.Queries.UNSAFECreateUser(context.Background(), pgtype.Text{String: mid, Valid: true})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating user",
				},
			})
			return
		}
		log.Println("was user created?", muser)

		exid, err := c.MatrixDB.Queries.UNSAFECreateExternalID(context.Background(), matrix_db.UNSAFECreateExternalIDParams{
			AuthProvider: u.Provider,
			ExternalID:   u.ID,
			UserID:       muser.String,
		})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating external id",
				},
			})
			return
		}
		log.Println("was external id created?", exid)

		err = c.MatrixDB.Queries.UNSAFECreateProfile(context.Background(), matrix_db.UNSAFECreateProfileParams{
			FullUserID: pgtype.Text{
				String: mid,
				Valid:  true,
			},
			UserID: u.Username,
		})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating profile",
				},
			})
			return
		}

		err = c.MatrixDB.Queries.UNSAFECreateUserDirectory(context.Background(), matrix_db.UNSAFECreateUserDirectoryParams{
			DisplayName: pgtype.Text{
				String: u.Username,
				Valid:  true,
			},
			UserID: mid,
		})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating user directory",
				},
			})
			return
		}

		did := RandomString(12)

		device_id, err := c.MatrixDB.Queries.UNSAFECreateDevice(context.Background(), matrix_db.UNSAFECreateDeviceParams{
			UserID:   muser.String,
			DeviceID: did,
		})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating device",
				},
			})
			return
		}
		log.Println("was device_id created?", device_id)
		access_token, err := c.MatrixDB.Queries.UNSAFECreateAccessToken(context.Background(), matrix_db.UNSAFECreateAccessTokenParams{
			UserID: muser.String,
			DeviceID: pgtype.Text{
				String: did,
				Valid:  true,
			},
			Token: RandomString(32),
		})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating access token",
				},
			})
			return
		}
		log.Println("was token created?", access_token)

		space_id, err := c.CreateUserSpace(muser.String, access_token, u.Username)
		if err != nil {
			log.Println(err)
		}

		token := RandomString(32)

		cr := time.Now().Unix()

		created, err := c.MatrixDB.Queries.GetUserCreatedAt(context.Background(), pgtype.Text{String: muser.String, Valid: true})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error getting user created at",
				},
			})
			return
		} else {
			cr = created.Int64
		}

		suser := &User{
			//UserID:   idu,
			Username:          u.Username,
			DisplayName:       u.Username,
			AccessToken:       token,
			MatrixAccessToken: access_token,
			MatrixUserID:      muser.String,
			MatrixDeviceID:    did,
			Age:               cr,
		}

		if space_id != nil {
			suser.UserSpaceID = *space_id
		}

		err = c.StoreUserSession(suser)

		// send success JSON
		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"created":      true,
				"access_token": token,
				"credentials":  suser,
			},
		})
		return

	} else {

		log.Println("user exists, we'll just log them in")

		log.Println(userID)
		log.Println(userID)
		log.Println(userID)
		log.Println(userID)

		did := RandomString(12)

		device_id, err := c.MatrixDB.Queries.UNSAFECreateDevice(context.Background(), matrix_db.UNSAFECreateDeviceParams{
			UserID:   userID,
			DeviceID: did,
		})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating device",
				},
			})
			return
		}
		log.Println("was device_id created?", device_id)
		access_token, err := c.MatrixDB.Queries.UNSAFECreateAccessToken(context.Background(), matrix_db.UNSAFECreateAccessTokenParams{
			UserID: userID,
			DeviceID: pgtype.Text{
				String: did,
				Valid:  true,
			},
			Token: RandomString(32),
		})
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "error creating access token",
				},
			})
			return
		}
		log.Println("was token created?", access_token)

		creds, err := c.MatrixDB.Queries.GetCredentials(context.Background(), pgtype.Text{
			String: strings.ToLower(u.Username),
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

		profile, err := c.MatrixDB.Queries.GetProfile(context.Background(), u.Username)
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

		room_alias := fmt.Sprintf("#@%s:%s", u.Username, c.Config.Matrix.PublicServer)
		creator := fmt.Sprintf("@%s:%s", u.Username, c.Config.Matrix.PublicServer)

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

		admin, err := c.MatrixDB.Queries.IsAdmin(context.Background(), pgtype.Text{String: u.ID, Valid: true})
		if err != nil {
			log.Println(err)
		}

		user := &User{
			//UserID:            idu,
			Username:          u.Username,
			Email:             creds.Email.String,
			DisplayName:       profile.Displayname.String,
			AvatarURL:         profile.AvatarUrl.String,
			AccessToken:       token,
			MatrixAccessToken: access_token,
			MatrixUserID:      userID,
			MatrixDeviceID:    did,
			UserSpaceID:       userspace,
			Age:               creds.CreatedAt.Int64,
			Verified:          creds.Verified,
			Admin:             admin,
		}

		err = c.StoreUserSession(user)

		spaces, err := c.MatrixDB.Queries.GetUserSpaces(context.Background(), pgtype.Text{String: userID, Valid: true})
		if err != nil {
			log.Println(err)
		}
		rooms, err := c.MatrixDB.Queries.GetJoinedRooms(context.Background(), pgtype.Text{String: userID, Valid: true})
		if err != nil {
			log.Println(err)
		}
		dms, err := c.MatrixDB.Queries.GetDMs(context.Background(), user.MatrixUserID)
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
				"dms":           dms,
			},
		})

	}
}

type GithubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type GithubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Bio   string `json:"bio"`
}

func (c *App) ValidateOauthGithub() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		code := query.Get("code")

		log.Println("recieved code ", code)

		if code == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "missing provider or code",
				},
			})
			return
		}

		// Retrieve the required parameters from the request body.
		clientID := ""
		clientSecret := ""

		for _, item := range c.Config.Oauth {
			if item.Provider == "github" && item.Enabled {
				clientID = item.ClientID
				clientSecret = item.ClientSecret
			}
		}

		if clientID == "" || clientSecret == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "this provider is not enabled",
				},
			})
			return
		}

		githubAPIEndpoint := "https://github.com/login/oauth/access_token"

		redirectURI := fmt.Sprintf("%s/oauth/github", c.Config.App.PublicDomain)

		data := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s&redirect_uri=%s", clientID, clientSecret, code, redirectURI)

		log.Println("url is ", data)

		req, err := http.NewRequest("POST", githubAPIEndpoint, strings.NewReader(data))
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Println(err, resp.StatusCode)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		var token GithubAccessTokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		log.Println("token is", token)

		if token.AccessToken == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "missing access token",
				},
			})
			return
		}

		identityEndpoint := "https://api.github.com/user"
		req, err = http.NewRequest("GET", identityEndpoint, nil)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		var user GithubUser
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			log.Println(err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"user": user,
			},
		})

	}
}
