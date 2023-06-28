package app

import (
	"context"
	"log"
	"net/http"

	matrix_db "shpong/db/matrix/gen"

	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) UpdateDisplayName() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type request struct {
			DisplayName string `json:"display_name"`
		}

		p, err := ReadRequestJSON(r, w, &request{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		log.Println("recieved payload ", p)
		user := c.LoggedInUser(r)

		err = c.MatrixDB.Queries.UpdateUserDirectoryDisplayName(context.Background(), matrix_db.UpdateUserDirectoryDisplayNameParams{
			UserID: user.MatrixUserID,
			DisplayName: pgtype.Text{
				String: p.DisplayName,
				Valid:  true,
			},
		})
		if err != nil {
			log.Println(err)
		}

		err = c.MatrixDB.Queries.UpdateProfilesDisplayName(context.Background(), matrix_db.UpdateProfilesDisplayNameParams{
			FullUserID: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			Displayname: pgtype.Text{
				String: p.DisplayName,
				Valid:  true,
			},
		})
		if err != nil {
			log.Println(err)
		}

		user.DisplayName = p.DisplayName

		err = c.StoreUserSession(user)
		if err != nil {
			log.Println(err)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"updated": true,
			},
		})
	}
}

func (c *App) UpdateAvatar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type request struct {
			URL string `json:"url"`
		}

		p, err := ReadRequestJSON(r, w, &request{})

		if err != nil {
			RespondWithBadRequestError(w)
			return
		}

		log.Println("recieved payload ", p)

		user := c.LoggedInUser(r)

		/*
			serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

			matrix, err := gomatrix.NewClient(serverName, user.MatrixUserID, user.MatrixAccessToken)
			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":   "Could not update avatar at this time.",
						"updated": false,
					},
				})
				return
			}

			err = matrix.SetAvatarURL(p.URL)
			if err != nil {
				log.Println(err)
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"error":   "Could not update avatar at this time.",
						"updated": false,
					},
				})
				return
			}
		*/

		err = c.MatrixDB.Queries.UpdateUserDirectoryAvatar(context.Background(), matrix_db.UpdateUserDirectoryAvatarParams{
			UserID: user.MatrixUserID,
			AvatarUrl: pgtype.Text{
				String: p.URL,
				Valid:  true,
			},
		})
		if err != nil {
			log.Println(err)
		}
		err = c.MatrixDB.Queries.UpdateProfilesAvatar(context.Background(), matrix_db.UpdateProfilesAvatarParams{
			FullUserID: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			AvatarUrl: pgtype.Text{
				String: p.URL,
				Valid:  true,
			},
		})
		if err != nil {
			log.Println(err)
		}

		user.AvatarURL = p.URL

		err = c.StoreUserSession(user)
		if err != nil {
			log.Println(err)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"updated": true,
			},
		})
	}
}
