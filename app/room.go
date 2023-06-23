package app

import (
	"context"
	"log"
	"net/http"
	"strings"

	matrix_db "shpong/db/matrix/gen"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) JoinSpace() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		space := chi.URLParam(r, "space")

		log.Println("what is space", space)

		if space == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "that space doesn't exist",
				},
			})
			return
		}

		alias := c.ConstructMatrixRoomID(space)

		user := c.LoggedInUser(r)

		// get space room_id and all it's children's room_ids
		sri, err := c.MatrixDB.Queries.GetSpaceRoomIDs(context.Background(), alias)

		if err != nil {
			log.Println("error getting space room ids: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "space does not exist",
					"exists": false,
				},
			})
			return
		}

		//log.Println(sri)

		matrix, err := c.NewMatrixClient(user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "internal server error",
				},
			})
			return
		}

		re, err := matrix.JoinRoom(sri.RoomID, "", nil)

		if err != nil {
			log.Println("could not join space", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error":   "inter server error",
					"message": err,
				},
			})
			return
		} else {
			log.Println(re)
		}

		alias = strings.ToLower(alias)

		details, err := c.MatrixDB.Queries.GetSpaceInfo(context.Background(), matrix_db.GetSpaceInfoParams{
			RoomAlias: alias,
			Creator: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
		})

		if err != nil {
			log.Println("error getting space info: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "space joined but could not get space details",
					"exists": false,
				},
			})
			return
		}

		//join all space child rooms
		/*
			go func() {
				for _, room_id := range sri.Rooms {

					re, err := matrix.JoinRoom(room_id, "", nil)

					if err != nil {
						log.Println("could not join room", err)
					}

					if re != nil {
						log.Println(re)
					}
				}
			}()
		*/

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"joined": true,
				"space":  details,
			},
		})

	}
}

func (c *App) JoinRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		room_id := query.Get("id")

		log.Println("what is room id", room_id)

		if room_id == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "internal server error",
				},
			})
			return
		}

		user := c.LoggedInUser(r)

		matrix, err := c.NewMatrixClient(user.MatrixUserID, user.MatrixAccessToken)
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "internal server error",
				},
			})
			return
		}

		re, err := matrix.JoinRoom(room_id, "", nil)

		if err != nil {
			log.Println("could not join room", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "inter server error",
				},
			})
			return
		}

		if re != nil {
			log.Println(re)
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"joined":  true,
				"room_id": re.RoomID,
			},
		})

	}
}

func (c *App) RoomJoined() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		room_id := query.Get("id")

		if room_id == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "internal server error",
				},
			})
			return
		}

		user := c.LoggedInUser(r)

		joined, err := c.MatrixDB.Queries.RoomJoined(context.Background(), matrix_db.RoomJoinedParams{
			UserID: pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			},
			RoomID: pgtype.Text{
				String: room_id,
				Valid:  true,
			},
		})

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": "inter server error",
				},
			})
			return
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"joined": joined,
			},
		})

	}
}
