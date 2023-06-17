package app

import (
	"fmt"
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
		// Extracting metadata using goquery
		doc := e.DOM

		log.Println("doc: ", doc.Html)

		// Get title
		title := doc.Find("title").Text()
		fmt.Println("Title:", title)
		lmd.Title = title

		// Get description
		description, err := doc.Find("meta[name='description']").Attr("content")
		if err {
			log.Println(err)
		}
		fmt.Println("Description:", description)
		lmd.Description = description

		// Get author
		author, _ := doc.Find("meta[name='author']").Attr("content")
		fmt.Println("Author:", author)
		lmd.Author = author

		// Get image metadata
		images := make([]string, 0)
		doc.Find("meta[property='og:image'], meta[name='twitter:image']").Each(func(_ int, s *goquery.Selection) {
			image, _ := s.Attr("content")
			images = append(images, image)
		})
		fmt.Println("Images:", strings.Join(images, ", "))
		if images != nil && len(images) > 0 {

			lmd.Image = images[0]
		}

		// Get OpenGraph or Twitter metadata
		ogMetadata := make(map[string]string)
		doc.Find("meta[property^='og:'], meta[name^='twitter:']").Each(func(_ int, s *goquery.Selection) {
			property, _ := s.Attr("property")
			name, _ := s.Attr("name")
			content, _ := s.Attr("content")
			if property != "" {
				ogMetadata[property] = content
			} else if name != "" {
				ogMetadata[name] = content
			}
		})
		fmt.Println("OpenGraph/Twitter Metadata:")
		for key, value := range ogMetadata {
			fmt.Printf("%s: %s\n", key, value)
		}
	})

	// Set error handler
	co.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	co.Visit(link)

	return lmd

}
