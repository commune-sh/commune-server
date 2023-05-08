package app

import (
	config "shpong/config"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis"
	"github.com/tidwall/buntdb"
)

type Cache struct {
	VerificationCodes *buntdb.DB
	Posts             *redis.Client
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
		Posts:             pdb,
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

	err := c.Cache.VerificationCodes.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(t.Email)
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
