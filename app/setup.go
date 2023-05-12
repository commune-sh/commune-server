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
		log.Println(err)
		return err
	}

	log.Println("registered user", resp)

	return nil
}
