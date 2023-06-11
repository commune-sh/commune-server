package app

import (
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
)

func (c *App) FetchLinkMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		href := query.Get("href")

		if href == "" {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "no link provided",
				},
			})
			return
		}

		metadata := c.Scrape(href)
		log.Println(metadata)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"metadata": metadata,
			},
		})

	}
}

type LinkMetaData struct {
	Title       string `json:"title",omitempty`
	Description string `json:"description",omitempty`
	Image       string `json:"image",omitempty`
	Author      string `json:"author",omitempty`
}

func (c *App) Scrape(link string) LinkMetaData {
	co := colly.NewCollector()
	extensions.RandomUserAgent(co)
	extensions.Referer(co)

	lmd := LinkMetaData{}

	co.OnHTML("head", func(e *colly.HTMLElement) {

		// Extract meta tags from the document
		metaTags := e.DOM.ParentsUntil("~").Find("meta")

		metaTags.Each(func(_ int, s *goquery.Selection) {
			// Search for og:type meta tags
			name, _ := s.Attr("name")
			prop, _ := s.Attr("property")

			if strings.EqualFold(name, "description") {
				description, _ := s.Attr("content")
				lmd.Description = description
			}

			if strings.EqualFold(name, "author") {
				author, _ := s.Attr("content")
				lmd.Author = author
			}

			if strings.EqualFold(prop, "og:image") {
				image, _ := s.Attr("content")
				lmd.Image = image
			}

			if lmd.Image == "" && strings.EqualFold(prop, "twitter:image:src") {
				image, _ := s.Attr("content")
				lmd.Image = image
			}

		})

	})

	co.OnHTML("head title", func(e *colly.HTMLElement) {
		lmd.Title = e.Text

	})

	co.Visit(link)

	return lmd

}
