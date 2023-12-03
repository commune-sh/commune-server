package app

import (
	"context"
	"log"
	"net/http"
)

func (c *App) HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data := map[string]any{
			"healthy":  true,
			"version":  c.Version[:7],
			"features": c.Config.Features,
			"restrictions": map[string]any{
				"space": c.Config.Restrictions.Space,
				"media": c.Config.Restrictions.Media,
			},
			"shortlink_server": c.Config.App.ShortlinkDomain,
		}

		oauth := make(map[string]any)

		for _, item := range c.Config.Oauth {
			if item.Enabled {
				oauth[item.Provider] = map[string]any{
					"client_id": item.ClientID,
				}
			}
		}

		if len(oauth) > 0 {
			data["oauth"] = oauth
		}

		if c.Config.ThirdParty.GIF.Enabled {
			data["gif"] = map[string]any{
				"enabled": true,
				"service": c.Config.ThirdParty.GIF.Service,
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: data,
		})

	}
}

func (c *App) Stats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		rows, err := c.MatrixDB.Queries.GetTablesRowCount(context.Background())

		if err != nil {

			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error":  "Couldn't get stats",
					"exists": false,
				},
			})
			return
		}

		type stats struct {
			Spaces int64 `json:"spaces"`
			Users  int64 `json:"users"`
		}

		log.Println("rows: ", rows)

		st := stats{}

		for _, row := range rows {
			if row.Table == "spaces" {
				st.Spaces = row.Rows
			}
			if row.Table == "users" {
				st.Users = row.Rows
			}
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: st,
		})

	}
}

func (c *App) HomeserverInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		co, err := c.MatrixDB.Queries.GetSpaceCount(context.Background())
		if err != nil {
			log.Println("error getting homeserver info: ", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Couldn't get homeserver info",
				},
			})
			return
		}

		ep, err := c.MatrixDB.Queries.GetFirstEvent(context.Background())
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Couldn't get homeserver info",
				},
			})
			return
		}

		data := map[string]any{
			"spaces": co.Spaces,
			"users":  co.Users,
			"since":  ep,
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: data,
		})

	}
}
