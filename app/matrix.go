package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"shpong/config"
	matrix_db "shpong/db/matrix/gen"
	"shpong/gomatrix"

	"github.com/Jeffail/gabs/v2"
)

type MatrixAccountResponse struct {
	Response    *gomatrix.RespRegister
	UserSpaceID string
}

func (c *App) CreateMatrixUserAccount(username, password string) (*MatrixAccountResponse, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, "", "")
	if err != nil {
		log.Println(err)
		return nil, err
	}

	server := fmt.Sprintf(`http://%s:%d`, c.Config.Matrix.Homeserver, c.Config.Matrix.Port)
	matrix.Prefix = "/_synapse/admin/v1"

	nonce, err := GetNonce(server)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	//actually register the user
	mac, err := ConstructMac(&NewUser{
		Username: username,
		Password: password,
		Admin:    false,
	}, nonce, c.Config.Auth.SharedSecret)
	if err != nil {
		log.Println(err)
		return nil, err
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
		return nil, err
	}

	matrix.SetCredentials(resp.UserID, resp.AccessToken)
	matrix.Prefix = "/_matrix/client/r0"

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
				"name": fmt.Sprintf(`@%s`, username),
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
		RoomAliasName: fmt.Sprintf(`@%s`, username),
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
	}

	log.Println("Was Room created?", crr)

	return &MatrixAccountResponse{
		Response:    resp,
		UserSpaceID: crr.RoomID,
	}, nil

}

func (c *App) NewMatrixClient(userID, accessToken string) (*gomatrix.Client, error) {

	serverName := c.URLScheme(c.Config.Matrix.Homeserver) + fmt.Sprintf(`:%d`, c.Config.Matrix.Port)

	matrix, err := gomatrix.NewClient(serverName, "", "")

	if accessToken != "" {
		matrix.SetCredentials(userID, accessToken)
	}

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return matrix, nil
}

func QueryMatrixServerHealth(c config.Matrix) {

	a := fmt.Sprintf(`http://%s:%d/_matrix/client/versions`, c.Homeserver, c.Port)

	resp, err := http.Get(a)
	if err != nil {
		panic(errors.New("Cannot connect to Matrix server."))
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	type Version struct {
		Versions         []string `json:"versions"`
		UnstableFeatures struct {
			OrgMatrixLabelBasedFiltering        bool `json:"org.matrix.label_based_filtering"`
			OrgMatrixE2ECrossSigning            bool `json:"org.matrix.e2e_cross_signing"`
			OrgMatrixMsc2432                    bool `json:"org.matrix.msc2432"`
			UkHalfShotMsc2666MutualRooms        bool `json:"uk.half-shot.msc2666.mutual_rooms"`
			IoElementE2EeForcedPublic           bool `json:"io.element.e2ee_forced.public"`
			IoElementE2EeForcedPrivate          bool `json:"io.element.e2ee_forced.private"`
			IoElementE2EeForcedTrustedPrivate   bool `json:"io.element.e2ee_forced.trusted_private"`
			OrgMatrixMsc3026BusyPresence        bool `json:"org.matrix.msc3026.busy_presence"`
			OrgMatrixMsc2285Stable              bool `json:"org.matrix.msc2285.stable"`
			OrgMatrixMsc3827Stable              bool `json:"org.matrix.msc3827.stable"`
			OrgMatrixMsc2716                    bool `json:"org.matrix.msc2716"`
			OrgMatrixMsc3440Stable              bool `json:"org.matrix.msc3440.stable"`
			OrgMatrixMsc3771                    bool `json:"org.matrix.msc3771"`
			OrgMatrixMsc3773                    bool `json:"org.matrix.msc3773"`
			FiMauMsc2815                        bool `json:"fi.mau.msc2815"`
			FiMauMsc2659                        bool `json:"fi.mau.msc2659"`
			OrgMatrixMsc3882                    bool `json:"org.matrix.msc3882"`
			OrgMatrixMsc3881                    bool `json:"org.matrix.msc3881"`
			OrgMatrixMsc3874                    bool `json:"org.matrix.msc3874"`
			OrgMatrixMsc3886                    bool `json:"org.matrix.msc3886"`
			OrgMatrixMsc3912                    bool `json:"org.matrix.msc3912"`
			OrgMatrixMsc3952IntentionalMentions bool `json:"org.matrix.msc3952_intentional_mentions"`
		} `json:"unstable_features"`
	}

	var v Version
	err = json.Unmarshal([]byte(body), &v)
	if err != nil {
		panic(err)
	}

}

func (c *App) ConstructMatrixID(username string) string {
	return fmt.Sprintf("@%s:%s", username, c.Config.Matrix.PublicServer)
}

func (c *App) ConstructMatrixUserRoomID(username string) string {
	return fmt.Sprintf("#@%s:%s", username, c.Config.Matrix.PublicServer)
}

func (c *App) ConstructMatrixRoomID(username string) string {
	return fmt.Sprintf("#%s:%s", username, c.Config.Matrix.PublicServer)
}

type sender struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	AvatarURL   string `json:"avatar_url"`
	DisplayName string `json:"display_name"`
}

type Event struct {
	Type           any      `json:"type"`
	Content        any      `json:"content"`
	Sender         sender   `json:"sender"`
	EventID        any      `json:"event_id"`
	StateKey       any      `json:"state_key,omitempty"`
	RoomAlias      any      `json:"room_alias,omitempty"`
	RoomID         any      `json:"room_id"`
	OriginServerTs any      `json:"origin_server_ts"`
	Unsigned       any      `json:"unsigned"`
	Slug           string   `json:"slug,omitempty"`
	ReplyCount     int64    `json:"reply_count"`
	Reactions      any      `json:"reactions,omitempty"`
	UserReactions  []string `json:"user_reactions,omitempty"`
}

type EventProcessor struct {
	JSON        *gabs.Container
	EventID     string
	Slug        string
	DisplayName string
	AvatarURL   string
	RoomAlias   string
	ReplyCount  int64
	Reactions   any
}

func ProcessComplexEvent(ep *EventProcessor) Event {

	e := Event{
		Type:    ep.JSON.Path("type").Data().(string),
		Content: ep.JSON.Path("content").Data().(any),
		Sender: sender{
			ID: ep.JSON.Path("sender").Data().(string),
		},
		RoomID:         ep.JSON.Path("room_id").Data().(string),
		OriginServerTs: ep.JSON.Path("origin_server_ts").Data().(any),
		Unsigned:       ep.JSON.Path("unsigned").Data().(any),
	}

	e.Sender.Username = GetLocalPart(e.Sender.ID)

	e.EventID = ep.EventID
	e.Slug = ep.Slug

	e.RoomAlias = ep.RoomAlias

	e.Sender.DisplayName = ep.DisplayName
	e.Sender.AvatarURL = ep.AvatarURL

	e.ReplyCount = ep.ReplyCount

	e.Reactions = ep.Reactions

	sk, ok := ep.JSON.Path("state_key").Data().(string)
	if ok {
		e.StateKey = sk
	}

	return e

}

func ProcessEvent(d *gabs.Container) Event {

	e := Event{
		Type:    d.Path("type").Data().(string),
		Content: d.Path("content").Data().(any),
		Sender: sender{
			ID: d.Path("sender").Data().(string),
		},
		RoomID:         d.Path("room_id").Data().(string),
		OriginServerTs: d.Path("origin_server_ts").Data().(any),
		Unsigned:       d.Path("unsigned").Data().(any),
	}

	sk, ok := d.Path("state_key").Data().(string)
	if ok {
		e.StateKey = sk
	}

	return e
}

type SpaceState struct {
	RoomID         string `json:"room_id"`
	Members        int64  `json:"members"`
	OriginServerTS int64  `json:"origin_server_ts"`
	Owner          string `json:"owner"`
	Space          any    `json:"space"`
	Children       any    `json:"children,omitempty"`
	Joined         bool   `json:"joined,omitempty"`
}

type state struct {
	Name   string `json:"name,omitempty"`
	Topic  string `json:"topic,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	Header string `json:"header,omitempty"`
}

func ProcessState(m matrix_db.GetSpaceStateRow) *SpaceState {

	var st state
	err := json.Unmarshal(m.State, &st)
	if err != nil {
		log.Println("Error unmarshalling state: ", err)
	}

	s := &SpaceState{
		RoomID:         m.RoomID,
		Members:        m.Members.Int64,
		OriginServerTS: m.OriginServerTS.Int64,
		Owner:          m.Owner.String,
		Space:          st,
		Children:       m.Children,
		Joined:         m.Joined,
	}

	return s
}
