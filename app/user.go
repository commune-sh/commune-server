package app

import (
	"log"
	"net/http"
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

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"updated": true,
			},
		})
	}
}
