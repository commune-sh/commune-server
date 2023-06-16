package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/hostrouter"
	"github.com/lpar/gzipped"
	"github.com/unrolled/secure"
)

func (c *App) Routes() {
	compressor := middleware.NewCompressor(5, "text/html", "text/css", "text/event-stream")
	compressor.SetEncoder("nop", func(w io.Writer, _ int) io.Writer {
		return w
	})

	// c.Router.Use(middleware.ThrottleBacklog(10, 50, time.Second*10))
	c.Router.Use(middleware.RequestID)
	c.Router.Use(middleware.RealIP)
	c.Router.Use(middleware.Logger)
	c.Router.Use(c.Recoverer)
	c.Router.Use(middleware.StripSlashes)
	c.Router.Use(compressor.Handler)

	c.CORS()
	c.ServeStaticFiles()

	hr := hostrouter.New()

	hr.Map(c.Config.App.Domain, routes(c))
	hr.Map(c.Config.App.SSRDomain, SSRDomain(c))
	// local dev please ignore
	hr.Map("192.168.1.12:8989", routes(c))

	c.Router.Mount("/", hr)
}

func SSRDomain(c *App) chi.Router {

	r := chi.NewRouter()

	compressor := middleware.NewCompressor(5, "text/html", "text/css")
	compressor.SetEncoder("nop", func(w io.Writer, _ int) io.Writer {
		return w
	})

	r.Use(compressor.Handler)
	r.Use(c.GetAuthSession)

	r.Get("/", c.Index())

	r.NotFound(c.NotFound)
	return r
}

func routes(c *App) chi.Router {
	sop := secure.Options{
		ContentSecurityPolicy: "script-src 'self' 'unsafe-eval' 'unsafe-inline' $NONCE",
		IsDevelopment:         false,
		AllowedHosts: []string{
			c.Config.App.Domain,
		},
	}

	secureMiddleware := secure.New(sop)

	r := chi.NewRouter()
	r.Use(c.GetAuthorizationToken)

	r.Route("/health_check", func(r chi.Router) {
		r.Get("/", c.HealthCheck())
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(secureMiddleware.Handler)
		r.Get("/", c.NotFound)
		r.Route("/signup", func(r chi.Router) {
			r.Use(secureMiddleware.Handler)
			r.Post("/verify/code", c.SendCode())
			r.Post("/verify", c.VerifyCode())
			r.Post("/", c.Signup())
		})
		r.Route("/username", func(r chi.Router) {
			r.Get("/", c.NotFound)
			r.Post("/exists", c.UsernameAvailable())
		})

		r.Route("/user", func(r chi.Router) {
			r.Post("/posts", c.UserPosts())
		})
	})

	r.Route("/account", func(r chi.Router) {
		r.Use(secureMiddleware.Handler)
		r.Post("/login", c.ValidateLogin())
		r.Post("/session", c.ValidateSession())
		r.Post("/token", c.ValidateToken())
		r.Post("/", c.CreateAccount())
		r.Route("/username", func(r chi.Router) {
			r.Get("/{username}", c.UsernameAvailable())
		})
	})

	r.Route("/feed", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Get("/", c.UserFeedEvents())
	})

	r.Route("/media", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Get("/presigned_url", c.GetPresignedURL())
	})

	r.Route("/default_spaces", func(r chi.Router) {
		r.Get("/", c.DefaultSpaces())
	})

	r.Route("/events", func(r chi.Router) {
		r.Get("/", c.AllEvents())
		//r.Get("/{room}", c.RoomEvents())
	})

	r.Route("/sync", func(r chi.Router) {
		r.Route("/", func(r chi.Router) {
			r.Get("/", c.SyncEvents())
		})
	})

	r.Route("/link", func(r chi.Router) {
		r.Get("/metadata", c.FetchLinkMetadata())
	})

	r.Route("/domain", func(r chi.Router) {
		r.Get("/{domain}/api", c.DomainAPIEndpoint())
	})

	r.Route("/search", func(r chi.Router) {
		r.Get("/{room_id}/events", c.SearchEvents())
	})

	r.Route("/event", func(r chi.Router) {
		r.Route("/", func(r chi.Router) {
			r.Use(c.RequireAuthentication)
			r.Post("/", c.CreatePost())
			r.Post("/redact", c.RedactPost())
			r.Post("/redact/reaction", c.RedactReaction())
			r.Put("/upvote", c.Upvote())
			r.Put("/downvote", c.Downvote())
		})
		r.Get("/{event}", c.Event())
		r.Get("/{event}/replies", c.EventReplies())
	})

	r.Route("/room", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Route("/joined", func(r chi.Router) {
			r.Get("/", c.RoomJoined())
		})
		r.Post("/join", c.JoinRoom())
	})
	r.Route("/space", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Post("/{space}/join", c.JoinSpace())
		r.Post("/create", c.CreateSpace())
	})

	r.Route("/{space}", func(r chi.Router) {
		r.Use(secureMiddleware.Handler)
		//r.Get("/post/{slug}", c.SpaceEvent())
		r.Get("/events", c.SpaceEvents())
		r.Get("/state", c.SpaceState())
		r.Get("/{room}/events", c.SpaceRoomEvents())
		r.Get("/{room}/messages", c.SpaceRoomMessages())
		//r.Get("/{room}/post/{slug}", c.SpaceEvent())
	})

	r.Route("/", func(r chi.Router) {
		r.Use(secureMiddleware.Handler)
		r.Get("/about", c.StaticPage())
		r.Get("/*", c.Index())
		// r.Get("/*", c.Dispatch())
	})

	compressor := middleware.NewCompressor(5, "text/html", "text/css")
	compressor.SetEncoder("nop", func(w io.Writer, _ int) io.Writer {
		return w
	})
	r.NotFound(c.NotFound)

	return r
}

func (c *App) NotFound(w http.ResponseWriter, r *http.Request) {

	RespondWithJSON(w, &JSONResponse{
		Code: http.StatusNotFound,
		JSON: map[string]any{
			"message": "resource not found",
		},
	})
}

func (c *App) OldNotFound(w http.ResponseWriter, r *http.Request) {
	us := c.LoggedInUser(r)
	type NotFoundPage struct {
		LoggedInUser interface{}
		Nonce        string
	}

	nonce := secure.CSPNonce(r.Context())
	pg := NotFoundPage{
		LoggedInUser: us,
		Nonce:        nonce,
	}
	c.Templates.ExecuteTemplate(w, "not-found", pg)
}

func (c *App) ServeStaticFiles() {

	path := "/static"
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "static")

	fs := http.StripPrefix(path, gzipped.FileServer(FileSystem{http.Dir(filesDir)}))

	if path != "/" && path[len(path)-1] != '/' {
		c.Router.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	c.Router.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=31536000")
		fs.ServeHTTP(w, r)
	}))
}

type FileSystem struct {
	fs http.FileSystem
}

func (nfs FileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if _, err := nfs.fs.Open(index); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (c *App) CORS() {
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"X-PINGOTHER", "Accept", "Authorization", "Image", "Attachment", "File-Type", "Content-Type", "X-CSRF-Token", "Access-Control-Allow-Origin"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	c.Router.Use(cors.Handler)
}

func (c *App) Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {

				logEntry := middleware.GetLogEntry(r)
				if logEntry != nil {
					logEntry.Panic(rvr, debug.Stack())
				} else {
					fmt.Fprintf(os.Stderr, "Panic: %+v\n", rvr)
					debug.PrintStack()
				}

				c.Error(w, r)
				return
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (c *App) Error(w http.ResponseWriter, r *http.Request) {
	us := c.LoggedInUser(r)

	type errorPage struct {
		LoggedInUser interface{}
		Nonce        string
	}

	nonce := secure.CSPNonce(r.Context())
	pg := errorPage{
		LoggedInUser: us,
		Nonce:        nonce,
	}

	c.Templates.ExecuteTemplate(w, "error", pg)
}

func (c *App) RoomTooLarge(w http.ResponseWriter, r *http.Request) {
	us := c.LoggedInUser(r)

	type errorPage struct {
		LoggedInUser interface{}
		Nonce        string
	}

	nonce := secure.CSPNonce(r.Context())
	pg := errorPage{
		LoggedInUser: us,
		Nonce:        nonce,
	}

	c.Templates.ExecuteTemplate(w, "room-too-large", pg)
}

func (c *App) StaticPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/*
				us := LoggedInUser(r)

				url := strings.TrimLeft(r.URL.Path, "/")

				type page struct {
					LoggedInUser interface{}
					Nonce        string
				}
				nonce := secure.CSPNonce(r.Context())

				pg := page{
					LoggedInUser: us,
					Nonce:        nonce,
				}
				c.Templates.ExecuteTemplate(w, url, pg)

			s := pgtype.UUID{}
			s.Scan("cd7b1316-f5f9-4f1d-b7e4-a0ac7515f26d")

			user, err := c.DB.Queries.GetUser(context.Background(), s)
			if err != nil {
				fmt.Fprintf(os.Stderr, "GetUser failed: %v\n", err)
			}

			fmt.Println(user)
		*/

		user, err := c.DB.Queries.GetUser(context.Background(), "testuser")
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetUser failed: %v\n", err)
		}
		fmt.Println(user.Username, user.Email)

		w.Write([]byte("lol"))
	}
}
