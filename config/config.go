package config

import (
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

type App struct {
	Domain          string `toml:"domain"`
	PublicDomain    string `toml:"public_domain"`
	SSRDomain       string `toml:"ssr_domain"`
	ShortlinkDomain string `toml:"shortlink_domain"`
	Port            int    `toml:"port"`
	CookieName      string `toml:"cookie_name"`
	SecureCookie    string `toml:"secure_cookie"`
	JWTKey          string `toml:"jwt_key"`
}

type Features struct {
	ShowIndex            bool `toml:"show_index" json:"show_index"`
	Social               bool `toml:"social" json:"social"`
	SpaceRooms           bool `toml:"space_rooms" json:"space_rooms"`
	RegistrationEnabled  bool `toml:"registration_enabled" json:"registration_enabled"`
	SpaceCreationEnabled bool `toml:"space_creation_enabled" json:"space_creation_enabled"`
	RequireEmail         bool `toml:"require_email" json:"require_email"`
}

type Matrix struct {
	Homeserver       string `toml:"homeserver"`
	FederationServer string `toml:"federation_server"`
	PublicServer     string `toml:"public_server"`
	Port             int    `toml:"port"`
	Password         string `toml:"password"`
	ConfigFile       string `toml:"config_file"`
}

type DB struct {
	Matrix string `toml:"matrix"`
}

type Redis struct {
	Address         string `toml:"address"`
	Password        string `toml:"password"`
	SessionsDB      int    `toml:"sessions_db"`
	PostsDB         int    `toml:"posts_db"`
	SystemDB        int    `toml:"system_db"`
	NotificationsDB int    `toml:"notifications_db"`
}

type Cache struct {
	IndexEvents  bool `toml:"index_events"`
	SpaceEvents  bool `toml:"space_events"`
	EventReplies bool `toml:"event_replies"`
}

type Authentication struct {
	VerifyEmail                bool   `toml:"verify_email"`
	DisableRegistration        bool   `toml:"disable_registration"`
	SharedSecret               string `toml:"shared_secret"`
	BlockPopularEmailProviders bool   `toml:"block_popular_email_providers"`
	QueryMXRecords             bool   `toml:"query_mx_records"`
}

type Privacy struct {
	DisablePublic bool `toml:"disable_public"`
}

type SMTP struct {
	Domain   string `toml:"domain"`
	Account  string `toml:"account"`
	Server   string `toml:"server"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type Storage struct {
	BucketName      string `toml:"bucket_name"`
	Region          string `toml:"region"`
	AccountID       string `toml:"account_id"`
	AccessKeyID     string `toml:"access_key_id"`
	AccessKeySecret string `toml:"access_key_secret"`
	Endpoint        string `toml:"endpoint"`
}

type Images struct {
	AccountID string `toml:"account_id"`
	APIToken  string `toml:"api_token"`
}

type ThirdParty struct {
	YoutubeKey string `toml:"youtube_key"`
	GIF        struct {
		Enabled  bool   `toml:"enabled"`
		Service  string `toml:"service"`
		Endpoint string `toml:"endpoint"`
		APIKey   string `toml:"api_key"`
	} `toml:"gif"`
}

type Discovery struct {
	Enabled bool   `toml:"enabled"`
	Server  string `toml:"server"`
	Key     string `toml:"key"`
	Domain  string `toml:"domain"`
}

type Restrictions struct {
	Space struct {
		RequireVerification        bool  `toml:"require_verification" json:"require_verification"`
		PrivateWithoutVerification bool  `toml:"private_without_verification" json:"private_without_verification"`
		SpacesPerUser              int   `toml:"spaces_per_user" json:"spaces_per_user"`
		TimeSinceLastSpace         int   `toml:"time_since_last_space" json:"time_since_last_space"`
		RejectReservedKeywords     bool  `toml:"reject_reserved_keywords" json:"reject_reserved_keywords"`
		SenderAge                  int32 `toml:"sender_age" json:"sender_age"`
	} `toml:"space" json:"space"`
	Media struct {
		VerifiedOnly bool `toml:"verified_only" json:"verified_only"`
		MaxSize      int  `toml:"max_size" json:"max_size"`
	}
}
type Search struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	APIKey  string `toml:"api_key"`
}

type Config struct {
	Name           string         `toml:"name"`
	Mode           string         `toml:"mode"`
	App            App            `toml:"app"`
	Matrix         Matrix         `toml:"matrix"`
	DB             DB             `toml:"db"`
	Redis          Redis          `toml:"redis"`
	Cache          Cache          `toml:"cache"`
	Authentication Authentication `toml:"authentication"`
	Privacy        Privacy        `toml:"privacy"`
	SMTP           SMTP           `toml:"smtp"`
	Features       Features       `toml:"features"`
	Storage        Storage        `toml:"storage"`
	Images         Images         `toml:"images"`
	ThirdParty     ThirdParty     `toml:"third_party"`
	Discovery      Discovery      `toml:"discovery"`
	Restrictions   Restrictions   `toml:"restrictions"`
	Search         Search         `toml:"search"`
}

var conf Config

// Read reads the config file and returns the Values struct
func Read(s string) (*Config, error) {
	file, err := os.Open(s)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	if _, err := toml.Decode(string(b), &conf); err != nil {
		panic(err)
	}

	return &conf, err
}
