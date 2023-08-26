package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type GIFData struct {
	Timestamp time.Time
	Data      interface{}
}

var GIFQueries = make(map[string]GIFData)

func (c *App) GetGIFCategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		url := fmt.Sprintf("%s/%s?key=%s", c.Config.ThirdParty.GIF.Endpoint, "categories", c.Config.ThirdParty.GIF.APIKey)

		if gd, ok := GIFQueries[url]; ok {

			now := time.Now()

			if now.Sub(gd.Timestamp).Minutes() < 60 {
				log.Println("time diff is", now.Sub(gd.Timestamp).Minutes())

				log.Println("sending gifs from cache")
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: gd.Data,
				})
				return
			}
		}

		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error sending GET request:", err)
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Error fetching Gifs",
				},
			})
			return
		}

		defer resp.Body.Close()

		type Response struct {
			Status string      `json:"status"`
			Data   interface{} `json:"data"`
		}

		responseBody, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Error fetching Gifs",
				},
			})
			return
		}

		data := json.RawMessage(responseBody)

		gd := GIFData{
			Timestamp: time.Now(),
			Data:      data,
		}

		GIFQueries[url] = gd

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: data,
		})

	}
}

func (c *App) GetGIFSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q := r.URL.Query()
		query := q.Get("q")
		limit := q.Get("limit")

		if query == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Query is required",
				},
			})
			return
		}

		if limit == "" {
			limit = "50"
		}

		sp := fmt.Sprintf("&q=%s&limit=%s", query, limit)

		url := fmt.Sprintf("%s/%s?key=%s%s", c.Config.ThirdParty.GIF.Endpoint, "search", c.Config.ThirdParty.GIF.APIKey, sp)

		if gd, ok := GIFQueries[url]; ok {

			now := time.Now()

			if now.Sub(gd.Timestamp).Minutes() < 60 {
				log.Println("time diff is", now.Sub(gd.Timestamp).Minutes())

				log.Println("sending gifs from cache")
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: gd.Data,
				})
				return
			}
		}
		resp, err := http.Get(url)
		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Error fetching Gifs",
				},
			})
			return
		}

		defer resp.Body.Close()

		type Response struct {
			Status string      `json:"status"`
			Data   interface{} `json:"data"`
		}

		responseBody, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "Error fetching Gifs",
				},
			})
			return
		}

		data := json.RawMessage(responseBody)

		gd := GIFData{
			Timestamp: time.Now(),
			Data:      data,
		}

		GIFQueries[url] = gd

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: data,
		})

	}
}
