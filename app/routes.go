package app

import (
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

	r.Route("/robots.txt", func(r chi.Router) {
		r.Get("/", c.RobotsTXT())
	})

	r.Route("/health_check", func(r chi.Router) {
		r.Get("/", c.HealthCheck())
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Put("/user/suspend", c.SuspendUser())
		r.Put("/event/pin", c.PinEventToIndex())
		r.Put("/event/unpin", c.UnpinIndexEvent())
	})

	r.Route("/account", func(r chi.Router) {
		r.Use(secureMiddleware.Handler)
		r.Post("/login", c.ValidateLogin())
		r.Get("/logout", c.Logout())
		r.Post("/session", c.ValidateSession())
		r.Post("/token", c.ValidateToken())
		r.Route("/password", func(r chi.Router) {
			r.Post("/", c.SendRecoveryCode())
			r.Post("/verify", c.VerifyRecoveryCode())
			r.Post("/reset", c.ResetPassword())
			r.Post("/update", c.UpdatePassword())
		})
		r.Route("/", func(r chi.Router) {
			r.Post("/verify/code", c.SendCode())
			r.Post("/verify", c.VerifyCode())
			r.Post("/verify/email", c.VerifyEmail())
			r.Route("/", func(r chi.Router) {
				r.Use(c.RequireAuthentication)
				r.Post("/display_name", c.UpdateDisplayName())
				r.Post("/avatar", c.UpdateAvatar())
			})
		})
		r.Route("/notifications", func(r chi.Router) {
			r.Get("/sync", c.SyncNotifications())
			r.Route("/", func(r chi.Router) {
				r.Use(c.RequireAuthentication)
				r.Get("/", c.GetNotifications())
				r.Put("/read", c.MarkRead())
			})
		})

		r.Post("/", c.CreateAccount())
		r.Route("/username", func(r chi.Router) {
			r.Get("/{username}", c.UsernameAvailable())
		})
		r.Route("/email", func(r chi.Router) {
			r.Get("/{email}", c.ValidateEmail())
		})
	})
	r.Route("/discover", func(r chi.Router) {
		r.Get("/", c.AllSpaces())
	})

	r.Route("/feed", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Get("/", c.UserFeedEvents())
	})

	r.Route("/media", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Get("/presigned_url", c.GetPresignedURL())
		r.Get("/upload_url", c.GetUploadURL())
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
			//r.Get("/", c.SyncEvents())
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
			r.Post("/state", c.CreateStateEvent())
			r.Post("/redact", c.RedactPost())
			r.Post("/redact/reaction", c.RedactReaction())
			r.Put("/upvote", c.Upvote())
			r.Put("/downvote", c.Downvote())
		})
		r.Get("/{event}", c.Event())
		r.Get("/{event}/replies", c.EventReplies())
	})

	r.Route("/room", func(r chi.Router) {
		r.Get("/{room}/messages", c.RoomMessages())
		r.Get("/{room}/sync", c.SyncMessages())
		r.Route("/", func(r chi.Router) {
			r.Use(c.RequireAuthentication)
			r.Route("/joined", func(r chi.Router) {
				r.Get("/", c.RoomJoined())
			})
			r.Post("/join", c.JoinRoom())
			r.Post("/leave", c.LeaveRoom())
		})
	})
	r.Route("/space", func(r chi.Router) {
		r.Use(c.RequireAuthentication)
		r.Post("/{space}/join", c.JoinSpace())
		r.Post("/{space}/leave", c.LeaveSpace())
		r.Post("/create", c.CreateSpace())
		r.Post("/room/create", c.CreateSpaceRoom())
	})

	r.Route("/{space}", func(r chi.Router) {
		r.Use(secureMiddleware.Handler)
		//r.Get("/post/{slug}", c.SpaceEvent())
		r.Get("/events", c.SpaceEvents())
		r.Get("/state", c.SpaceState())
		r.Route("/power_levels", func(r chi.Router) {
			r.Use(c.RequireAuthentication)
			r.Get("/", c.GetPowerLevels())
		})
		r.Get("/{room}/events", c.SpaceRoomEvents())
		//r.Get("/{room}/post/{slug}", c.SpaceEvent())
	})

	r.Route("/", func(r chi.Router) {
		r.Use(secureMiddleware.Handler)
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
