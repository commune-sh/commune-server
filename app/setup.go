package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"shpong/gomatrix"
	"shpong/static"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

var BannedEmails []string
var ReservedKeywords []string

func BuildEmailBanlist() {

	domains, err := static.Files.ReadFile("emails.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(domains, &BannedEmails)
}

func BuildReservedKeywordsList() {

	reserved, err := static.Files.ReadFile("reserved.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(reserved, &ReservedKeywords)
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

func IsKeywordReserved(keyword string) bool {
	for _, word := range ReservedKeywords {
		if word == keyword {
			return true
		}
	}
	return false
}

func InitViews(db *MatrixDB) {
	var exists bool
	err := db.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_matviews WHERE matviewname = $1)", "aliases").Scan(&exists)
	if err != nil {
		log.Fatal("Error checking for materialized view existence:", err)
	}
	if !exists {
		log.Println("initiating views")
		MakeViews(db)
	}
}

func MakeViews(db *MatrixDB) {
	log.Println("making views")

	dir := "db/matrix/views"

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {

		if strings.HasSuffix(file.Name(), ".sql") {
			log.Println("Processing SQL file:", file.Name())

			filePath := filepath.Join(dir, file.Name())
			sqlContent, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading SQL file %s: %v\n", file.Name(), err)
				continue
			}

			sqlString := string(sqlContent)

			tx, err := db.Exec(context.Background(), sqlString)
			if err != nil {
				log.Println(err)
			}
			log.Println(tx)
		}
	}
}

func (c *App) Setup() {
	log.Println("setting up app")
	exists, err := c.SetupDefaultMatrixAccount()
	if err != nil {
		log.Println(err)
	}
	if exists {
		c.DefaultMatrixAccount = fmt.Sprintf(`@%s:%s`, c.Config.Name, c.Config.Matrix.PublicServer)
	}

	room_id, err := c.SetupPublicSpace()
	if err != nil {
		log.Println(err)
	}

	if room_id != "" && len(room_id) > 0 {
		c.DefaultMatrixSpace = room_id
	}

	log.Println(c.DefaultMatrixAccount, c.DefaultMatrixSpace)
	/*
		err = c.SetupDefaultSpaces()
		if err != nil {
			log.Println(err)
		}
	*/
}

func (c *App) SetupDefaultMatrixAccount() (bool, error) {

	user := fmt.Sprintf(`@%s:%s`, c.Config.Name, c.Config.Matrix.PublicServer)

	exists, err := c.MatrixDB.Queries.DoesMatrixUserExist(context.Background(), pgtype.Text{String: user, Valid: true})

	if err != nil {
		log.Println(err)
		return false, err
	}

	if exists {
		return true, nil
	}

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, "", "")
	if err != nil {
		log.Println(err)
		return false, err
	}

	username := c.Config.Name
	password := c.Config.Matrix.Password

	server := fmt.Sprintf(`http://%s:%d`, c.Config.Matrix.Homeserver, c.Config.Matrix.Port)
	matrix.Prefix = "/_synapse/admin/v1"

	nonce, err := GetNonce(server)
	if err != nil {
		log.Println(err)
		return false, err
	}

	//actually register the user
	mac, err := ConstructMac(&NewUser{
		Username: username,
		Password: password,
		Admin:    false,
	}, nonce, c.Config.Authentication.SharedSecret)
	if err != nil {
		log.Println(err)
		return false, err
	}

	req := &gomatrix.ReqLegacyRegister{
		Username: username,
		Password: password,
		Type:     "org.matrix.login.shared_secret",
		Mac:      mac,
		Admin:    false,
		Nonce:    nonce,
	}

	resp, _, err := matrix.LegacyRegister(req)

	if err != nil || resp == nil {
		return false, err
	}

	log.Println("registered default matrix user", resp)

	return true, nil
}

func (c *App) SetupPublicSpace() (string, error) {

	alias := fmt.Sprintf(`#%s:%s`, c.Config.Name, c.Config.Matrix.PublicServer)

	room_id, err := c.MatrixDB.Queries.DoesDefaultSpaceExist(context.Background(), alias)

	if room_id != "" && len(room_id) > 0 {
		return room_id, nil
	}

	matrix, resp, err := c.DefaultMatrixClient()
	if err != nil {
		log.Println(err)
		return "", err
	}

	pl := gomatrix.Event{
		Type: "m.room.power_levels",
		Content: map[string]interface{}{
			"ban": 60,
			"events": map[string]interface{}{
				"m.room.name":         60,
				"m.room.power_levels": 100,
				"m.room.create":       10,
				"m.space.child":       10,
				"m.space.parent":      10,
			},
			"events_default": 10,
			"invite":         10,
			"kick":           60,
			"notifications": map[string]interface{}{
				"room": 20,
			},
			"redact":        10,
			"state_default": 10,
			"users": map[string]interface{}{
				resp.UserID: 100,
			},
			"users_default": 10,
		},
	}

	initState := []gomatrix.Event{
		gomatrix.Event{
			Type: "m.room.history_visibility",
			Content: map[string]interface{}{
				"history_visibility": "world_readable",
			},
		}, gomatrix.Event{
			Type: "m.room.guest_access",
			Content: map[string]interface{}{
				"guest_access": "can_join",
			},
		}, gomatrix.Event{
			Type: "m.room.name",
			Content: map[string]interface{}{
				"name": c.Config.Name,
			},
		}, gomatrix.Event{
			Type: "m.space.default",
			Content: map[string]interface{}{
				"default": true,
			},
		},
		pl,
	}

	creq := &gomatrix.ReqCreateRoom{
		RoomAliasName: c.Config.Name,
		Preset:        "public_chat",
		Visibility:    "public",
		CreationContent: map[string]interface{}{
			"type": "m.space",
		},
		InitialState: initState,
	}

	crr, err := matrix.CreateRoom(creq)

	if err != nil || crr == nil {
		log.Println(err)
		return "", err
	}

	log.Println("Was default space created?", crr)

	return crr.RoomID, nil
}

func (c *App) SetupDefaultSpaces() error {

	matrix, resp, err := c.DefaultMatrixClient()
	if err != nil {
		log.Println(err)
		return err
	}
	//spaces = []string{"animals", "art", "books", "business", "cars", "celebrities", "comics", "culture", "education", "entertainment", "fashion", "food", "gaming", "health", "howto", "humor", "internet", "lifestyle", "movies", "music", "news", "nsfw", "parenting", "politics", "religion", "science", "space", "sports", "technology", "travel", "other"}

	spaces := []string{c.Config.Name}

	for _, space := range spaces {

		alias := fmt.Sprintf(`#%s:%s`, space, c.Config.Matrix.PublicServer)

		exists, err := c.MatrixDB.Queries.DoesSpaceExist(context.Background(), alias)

		if err != nil {
			return err
		} else if exists {
			return nil
		}

		pl := gomatrix.Event{
			Type: "m.room.power_levels",
			Content: map[string]interface{}{
				"ban": 60,
				"events": map[string]interface{}{
					"m.room.name":         60,
					"m.room.power_levels": 100,
					"m.room.create":       10,
					"m.space.child":       10,
					"m.space.parent":      10,
				},
				"events_default": 10,
				"invite":         10,
				"kick":           60,
				"notifications": map[string]interface{}{
					"room": 20,
				},
				"redact":        10,
				"state_default": 10,
				"users": map[string]interface{}{
					resp.UserID: 100,
				},
				"users_default": 10,
			},
		}

		initState := []gomatrix.Event{
			gomatrix.Event{
				Type: "m.room.history_visibility",
				Content: map[string]interface{}{
					"history_visibility": "world_readable",
				},
			}, gomatrix.Event{
				Type: "m.room.guest_access",
				Content: map[string]interface{}{
					"guest_access": "can_join",
				},
			}, gomatrix.Event{
				Type: "m.room.name",
				Content: map[string]interface{}{
					"name": space,
				},
			}, gomatrix.Event{
				Type: "m.space.default",
				Content: map[string]interface{}{
					"default": true,
				},
			},
			pl,
		}

		creq := &gomatrix.ReqCreateRoom{
			RoomAliasName: space,
			Preset:        "public_chat",
			Visibility:    "public",
			CreationContent: map[string]interface{}{
				"type": "m.space",
			},
			InitialState: initState,
		}

		crr, err := matrix.CreateRoom(creq)

		if err != nil || crr == nil {
			log.Println(err)
			return err
		}

		log.Println("Was space created?", crr)

	}

	return nil
}

func (c *App) DefaultMatrixClient() (*gomatrix.Client, *gomatrix.RespLogin, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, "", "")
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}

	username := c.Config.Name
	password := c.Config.Matrix.Password

	resp, err := matrix.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     username,
		Password: password,
	})

	if resp != nil {
		matrix.SetCredentials(resp.UserID, resp.AccessToken)
		matrix.Prefix = "/_matrix/client/r0"
	}

	return matrix, resp, nil

}
