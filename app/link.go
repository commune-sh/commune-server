package app

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
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

		if strings.Contains(href, "yout") {
			lmd, err := c.GetYoutubeMetadata(href)
			if err == nil && lmd != nil {
				RespondWithJSON(w, &JSONResponse{
					Code: http.StatusOK,
					JSON: map[string]any{
						"metadata": lmd,
					},
				})
			}
			return
		}

		metadata := c.Scrape(href)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: map[string]any{
				"metadata": metadata,
			},
		})

	}
}

func (c *App) GetYoutubeMetadata(href string) (*LinkMetaData, error) {

	up, err := url.Parse(href)
	if err != nil || up == nil || up.Host == "" {
		log.Println(err)
	}

	isYoutube := up.Host == "www.youtube.com" || up.Host == "youtube.com"
	isShortYoutube := up.Host == "youtu.be"

	if isYoutube || isShortYoutube && len(c.Config.ThirdParty.YoutubeKey) > 0 {
		log.Println("is youtube!")

		m, err := url.ParseQuery(up.RawQuery)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		lmd := &LinkMetaData{}

		if isYoutube {
			if len(m) > 0 {
				lmd.YoutubeID = m["v"][0]
			}
		} else if isShortYoutube {

			if isShortYoutube {
				lmd.YoutubeID = up.Path[1:]
			}
		}

		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, 7*time.Second)
		service, err := youtube.NewService(ctx, option.WithAPIKey(c.Config.ThirdParty.YoutubeKey))
		if err != nil {
			log.Println(err)
			return nil, err
		}

		videos := service.Videos.List([]string{"id", "snippet"})

		videos = videos.Id(lmd.YoutubeID)

		response, err := videos.Do()

		if err != nil {
			log.Println(err)
			return nil, err
		}

		if response != nil {

			items := response.Items

			log.Println(items)

			if items != nil && len(items) >= 1 {
				lmd.Title = items[0].Snippet.Title
				lmd.Description = items[0].Snippet.Description
				lmd.Image = items[0].Snippet.Thumbnails.Default.Url
				lmd.Author = items[0].Snippet.ChannelTitle
			}

			if len(lmd.Description) > 100 {
				lmd.Description = lmd.Description[:100]
			}

			return lmd, nil

		}
	}
	return nil, err
}

type LinkMetaData struct {
	Title       string `json:"title",omitempty`
	Description string `json:"description",omitempty`
	Image       string `json:"image",omitempty`
	Author      string `json:"author",omitempty`
	YoutubeID   string `json:"youtube_id",omitempty`
}

func (c *App) Scrape(link string) LinkMetaData {
	co := colly.NewCollector()
	extensions.RandomUserAgent(co)
	extensions.Referer(co)

	lmd := LinkMetaData{}

	co.OnHTML("head", func(e *colly.HTMLElement) {
		// Extracting metadata using goquery
		doc := e.DOM

		// Get title
		title := doc.Find("title").Text()
		lmd.Title = title

		// Get description
		description, err := doc.Find("meta[name='description']").Attr("content")
		if err {
			log.Println(err)
		}
		lmd.Description = description

		// Get author
		author, _ := doc.Find("meta[name='author']").Attr("content")
		lmd.Author = author

		// Get image metadata
		images := make([]string, 0)
		doc.Find("meta[property='og:image'], meta[name='twitter:image']").Each(func(_ int, s *goquery.Selection) {
			image, _ := s.Attr("content")
			images = append(images, image)
		})
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
	})

	// Set error handler
	co.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	co.Visit(link)

	return lmd

}
