package app

import (
	"encoding/json"
	"log"
	config "shpong/config"
	"time"

	"github.com/go-redis/redis"
	"github.com/tidwall/buntdb"
)

type Cache struct {
	VerificationCodes *buntdb.DB
	Events            *redis.Client
}

func NewCache(conf *config.Config) (*Cache, error) {

	pdb := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Address,
		Password: conf.Redis.Password,
		DB:       conf.Redis.PostsDB,
	})

	db, err := buntdb.Open(":memory:")
	if err != nil {
		panic(err)
	}

	c := &Cache{
		VerificationCodes: db,
		Events:            pdb,
	}

	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("mykey", "myvalue", nil)
		return err
	})

	return c, nil
}

func (c *App) AddCodeToCache(key string, t any) error {

	serialized, err := json.Marshal(t)
	if err != nil {
		log.Println(err)
		return err
	}

	err = c.Cache.VerificationCodes.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(serialized), &buntdb.SetOptions{Expires: true, TTL: time.Minute * 60})
		return err
	})
	log.Println("added to cache: ", key, t)
	return nil
}

func (c *App) DoesEmailCodeExist(t *CodeVerification) (bool, error) {

	exists := false

	log.Println("verifying code: ", t)

	err := c.Cache.VerificationCodes.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(t.Session)
		if err != nil {
			log.Println(err)
			return err
		}

		var d CodeVerification
		err = json.Unmarshal([]byte(val), &d)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Println("found in cache: ", d)

		if d.Session == t.Session && d.Code == t.Code && d.Email == t.Email {
			exists = true
		}

		return nil
	})

	if err != nil {
		return exists, err
	}

	return exists, nil
}
