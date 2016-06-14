package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/raintank/worldping-api/pkg/log"
)

var (
	validTTL   = time.Minute * 5
	invalidTTL = time.Second * 30
	cache      *AuthCache
)

type AuthCache struct {
	sync.RWMutex
	items map[string]CacheItem
}

type CacheItem struct {
	User       *SignedInUser
	ExpireTime time.Time
}

func (a *AuthCache) Get(key string) (*SignedInUser, bool) {
	a.RLock()
	defer a.RUnlock()
	if c, ok := a.items[key]; ok && c.ExpireTime.After(time.Now()) {
		return c.User, true
	}
	return nil, false
}

func (a *AuthCache) Set(key string, u *SignedInUser, ttl time.Duration) {
	a.Lock()
	a.items[key] = CacheItem{
		User:       u,
		ExpireTime: time.Now().Add(ttl),
	}
	a.Unlock()
}

func init() {
	cache = &AuthCache{items: make(map[string]CacheItem)}
}

func Auth(adminKey, keyString string) (*SignedInUser, error) {
	if keyString == adminKey {
		return &SignedInUser{
			Role:    ROLE_ADMIN,
			OrgId:   1,
			OrgName: "Admin",
			OrgSlug: "admin",
			IsAdmin: true,
			key:     keyString,
		}, nil
	}

	// check the cache
	log.Debug("Checking cache for apiKey")
	user, cached := cache.Get(keyString)
	if user != nil {
		log.Debug("valid key cached")
		return user, nil
	}
	if cached {
		log.Debug("invalid key cached")
		return nil, ErrInvalidApiKey
	}

	//validate the API key against grafana.net
	payload := url.Values{}
	payload.Add("token", keyString)
	res, err := http.PostForm("https://grafana.net/api/api-keys/check", payload)

	if err != nil {
		log.Error(3, "failed to check apiKey. %s", err)
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	log.Debug("apiKey check response was: %s", body)
	res.Body.Close()
	if res.StatusCode != 200 {
		//add the invalid key to the cache
		log.Debug("Caching invalidKey response for %d seconds", invalidTTL/time.Second)
		cache.Set(keyString, nil, invalidTTL)

		return nil, ErrInvalidApiKey
	}

	user = &SignedInUser{key: keyString}
	err = json.Unmarshal(body, user)
	if err != nil {
		log.Error(3, "failed to parse api-keys/check response. %s", err)
		return nil, err
	}

	// add the user to the cache.
	log.Debug("Caching validKey response for %d seconds", validTTL/time.Second)
	cache.Set(keyString, user, validTTL)
	return user, nil
}
