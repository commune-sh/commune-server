package app

import (
	"context"
	"log"
	"net/http"
	matrix_db "shpong/db/matrix/gen"
	"strconv"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *App) SpaceRoomMessages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		space := chi.URLParam(r, "space")
		room := chi.URLParam(r, "room")

		log.Println("space is", space)
		log.Println("room is", room)

		alias := c.ConstructMatrixRoomID(space)

		ssp := matrix_db.GetSpaceStateParams{
			RoomAlias: alias,
		}

		if user != nil && user.MatrixUserID != "" {
			ssp.UserID = pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			}
		}

		// check if space exists in DB
		state, err := c.MatrixDB.Queries.GetSpaceState(context.Background(), ssp)

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "space does not exist",
					"exists": false,
				},
			})
			return
		}

		sps := ProcessState(state)

		scp := matrix_db.GetSpaceChildParams{
			ParentRoomAlias: pgtype.Text{
				String: alias,
				Valid:  true,
			},
			ChildRoomAlias: pgtype.Text{
				String: room,
				Valid:  true,
			},
		}

		if user != nil {
			log.Println("user is ", user.MatrixUserID)
			scp.UserID = pgtype.Text{
				String: user.MatrixUserID,
				Valid:  true,
			}
		}

		crs, err := c.MatrixDB.Queries.GetSpaceChild(context.Background(), scp)

		if err != nil || crs.ChildRoomID.String == "" {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "space room does not exist",
					"state":  sps,
					"exists": false,
				},
			})
			return
		}
		log.Println("what is child room ID?", crs.ChildRoomID)

		// get events for this space
		events, err := c.GetSpaceMessages(&SpaceMessagesParams{
			RoomID: crs.ChildRoomID.String,
			Last:   r.URL.Query().Get("last"),
			After:  r.URL.Query().Get("after"),
			Topic:  r.URL.Query().Get("topic"),
		})

		if err != nil {
			log.Println("error getting event: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusInternalServerError,
				JSON: map[string]any{
					"error": err,
				},
			})
			return
		}

		// get events for this space

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"events": events,
			},
		})

	}
}

type SpaceMessagesParams struct {
	RoomID string
	Last   string
	After  string
	Topic  string
}

func (c *App) GetSpaceMessages(p *SpaceMessagesParams) (*[]Event, error) {

	sreq := matrix_db.GetSpaceMessagesParams{
		RoomID: p.RoomID,
	}

	if len(p.Topic) > 0 {
		sreq.Topic = pgtype.Text{
			String: p.Topic,
			Valid:  true,
		}
	}

	// get events for this space

	if p.Last != "" {
		i, _ := strconv.ParseInt(p.Last, 10, 64)
		log.Println("adding last", i)
		sreq.OriginServerTS = pgtype.Int8{
			Int64: i,
			Valid: true,
		}
	}

	if p.After != "" {
		i, _ := strconv.ParseInt(p.After, 10, 64)
		log.Println(i)
	}

	// get events for this space
	events, err := c.MatrixDB.Queries.GetSpaceMessages(context.Background(), sreq)

	if err != nil {
		log.Println("error getting event: ", err)
		return nil, err
	}

	items := []Event{}

	for _, item := range events {

		json, err := gabs.ParseJSON([]byte(item.JSON.String))
		if err != nil {
			log.Println("error parsing json: ", err)
		}

		s := ProcessComplexEvent(&EventProcessor{
			EventID:     item.EventID,
			Slug:        item.Slug,
			JSON:        json,
			RoomAlias:   item.RoomAlias.String,
			DisplayName: item.DisplayName.String,
			AvatarURL:   item.AvatarUrl.String,
			ReplyCount:  item.Replies,
			Reactions:   item.Reactions,
			Edited:      item.Edited,
			EditedOn:    item.EditedOn,
		})

		items = append(items, s)
	}

	return &items, nil
}
