package config

import (
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

type App struct {
	Domain          string `toml:"domain"`
	ShortlinkDomain string `toml:"shortlink_domain"`
	Port            int    `toml:"port"`
	CookieName      string `toml:"cookie_name"`
	SecureCookie    string `toml:"secure_cookie"`
	JWTKey          string `toml:"jwt_key"`
}

type Features struct {
	Social     bool `toml:"social"`
	SpaceRooms bool `toml:"space_rooms"`
}

type Matrix struct {
	Homeserver       string `toml:"homeserver"`
	FederationServer string `toml:"federation_server"`
	PublicServer     string `toml:"public_server"`
	Port             int    `toml:"port"`
	Password         string `toml:"password"`
}

type DB struct {
	App    string `toml:"app"`
	Matrix string `toml:"matrix"`
}

type Redis struct {
	Address    string `toml:"address"`
	Password   string `toml:"password"`
	SessionsDB int    `toml:"sessions_db"`
	PostsDB    int    `toml:"posts_DB"`
}

type Cache struct {
	IndexEvents bool `toml:"index_events"`
}

type Auth struct {
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

type Tenor struct {
	Key string `toml:"key"`
}

type Config struct {
	Name       string  `toml:"name"`
	Mode       string  `toml:"mode"`
	App        App     `toml:"app"`
	Matrix     Matrix  `toml:"matrix"`
	DB         DB      `toml:"db"`
	Redis      Redis   `toml:"redis"`
	Cache      Cache   `toml:"cache"`
	YoutubeKey string  `toml:"youtube_key"`
	Auth       Auth    `toml:"auth"`
	Privacy    Privacy `toml:"privacy"`
	SMTP       SMTP    `toml:"smtp"`
	Tenor      Tenor   `toml:"tenor"`
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
