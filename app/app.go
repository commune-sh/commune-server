package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"shpong/config"
	"shpong/static"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi"
	"github.com/go-redis/redis"
	"github.com/gorilla/sessions"
	"github.com/robfig/cron/v3"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type App struct {
	Config               *config.Config
	Router               *chi.Mux
	HTTP                 *http.Server
	Templates            *Template
	Sessions             *sessions.CookieStore
	SessionsStore        *redis.Client
	DB                   *DB
	MatrixDB             *MatrixDB
	Cron                 *cron.Cron
	Cache                *Cache
	MediaStorage         *s3.Client
	DefaultMatrixAccount string
	DefaultMatrixSpace   string
	Version              string
}

func (c *App) Activate() {
	log.Println("Started App.")

	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint

		if err := c.HTTP.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
			log.Printf("Shutdown by user")
		}
		close(idleConnsClosed)
	}()

	if err := c.HTTP.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}

type StartRequest struct {
	Config string
}

var CONFIG_FILE string
var PRODUCTION_MODE bool
var AssetFiles map[string]string

func Start(s *StartRequest) {

	CONFIG_FILE = s.Config

	conf, err := config.Read(s.Config)
	if err != nil {
		panic(err)
	}

	PRODUCTION_MODE = conf.Mode == "production"

	db, err := NewDB()
	if err != nil {
		panic(err)
	}

	mdb, err := NewMatrixDB()
	if err != nil {
		panic(err)
	}

	QueryMatrixServerHealth(conf.Matrix)

	tmpl, err := NewTemplate()
	if err != nil {
		panic(err)
	}

	AssetFiles, err = BuildTemplateAssets()

	router := chi.NewRouter()

	redis := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Address,
		Password: conf.Redis.Password,
		DB:       conf.Redis.SessionsDB,
	})

	sess := NewSession(conf.App.SecureCookie)
	sess.Options.Domain = fmt.Sprintf(`.%s`, conf.App.Domain)

	cron := cron.New()

	cache, err := NewCache(conf)
	if err != nil {
		panic(err)
	}

	BuildEmailBanlist()

	server := &http.Server{
		ReadTimeout:       5 * time.Minute,
		ReadHeaderTimeout: 30 * time.Second,
		//WriteTimeout: 60 * time.Second,
		IdleTimeout: 120 * time.Second,
		Addr:        fmt.Sprintf(`:%d`, conf.App.Port),
		Handler:     router,
	}

	c := &App{
		DB:            db,
		MatrixDB:      mdb,
		Config:        conf,
		HTTP:          server,
		Router:        router,
		Templates:     tmpl,
		SessionsStore: redis,
		Sessions:      sess,
		Cron:          cron,
		Cache:         cache,
	}

	media, err := c.NewMediaStorage()
	if err != nil {
		panic(err)
	}
	c.MediaStorage = media

	c.Version = func() string {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					return setting.Value
				}
			}
		}
		return ""
	}()

	c.Middleware()
	c.Routes()

	// c.Build()

	c.Setup()

	// go c.Cron.AddFunc("*/15 * * * *", c.RefreshCache)
	// go c.Cron.Start()

	c.UpdateIndexEventsCache()

	c.Activate()
}

var BannedEmails []string

func BuildEmailBanlist() {

	domains, err := static.Files.ReadFile("emails.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(domains, &BannedEmails)
}

func IsEmailBanned(email string) bool {
	// strip email domain from email
	email = email[strings.LastIndex(email, "@")+1:]

	for _, domain := range BannedEmails {
		if domain == email {
			return true
		}
	}
	return false
}

func (c *App) RefreshApp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.Config.Mode != "production" {
			log.Println("PRODUCTION MODE")
			w.Write([]byte(RandomString(64)))
			return
		}
		w.Write([]byte(RandomString(64)))
	}
}
