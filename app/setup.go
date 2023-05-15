package app

import (
	"fmt"
	"log"
	"shpong/gomatrix"
)

func (c *App) Setup() {
	log.Println("setting up app")
	err := c.SetupDefaultMatrixAccount()
	if err != nil {
		log.Println(err)
	}
	err = c.SetupDefaultSpaces()
	if err != nil {
		log.Println(err)
	}
}

func (c *App) SetupDefaultMatrixAccount() error {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, "", "")
	if err != nil {
		log.Println(err)
		return err
	}

	username := c.Config.App.Domain
	password := c.Config.Matrix.Password

	server := fmt.Sprintf(`http://%s:%d`, c.Config.Matrix.Homeserver, c.Config.Matrix.Port)
	matrix.Prefix = "/_synapse/admin/v1"

	nonce, err := GetNonce(server)
	if err != nil {
		log.Println(err)
		return err
	}

	//actually register the user
	mac, err := ConstructMac(&NewUser{
		Username: username,
		Password: password,
		Admin:    false,
	}, nonce, c.Config.Auth.SharedSecret)
	if err != nil {
		log.Println(err)
		return err
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
		return err
	}

	log.Println("registered default matrix user", resp)

	return nil
}

func (c *App) SetupDefaultSpaces() error {

	matrix, resp, err := c.DefaultMatrixClient()
	if err != nil {
		log.Println(err)
		return err
	}
	//spaces = []string{"animals", "art", "books", "business", "cars", "celebrities", "comics", "culture", "education", "entertainment", "fashion", "food", "gaming", "health", "howto", "humor", "internet", "lifestyle", "movies", "music", "news", "nsfw", "parenting", "politics", "religion", "science", "space", "sports", "technology", "travel", "other"}

	spaces := []string{"zoink", "boink", "loink"}

	for _, space := range spaces {

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
				Type: "shpong.room",
				Content: map[string]interface{}{
					"room_type": "profile",
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

	username := c.Config.App.Domain
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
