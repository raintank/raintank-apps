package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/grafana/grafana/pkg/log"
)

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
		return nil, ErrInvalidApiKey
	}

	user := &SignedInUser{key: keyString}
	err = json.Unmarshal(body, user)
	if err != nil {
		log.Error(3, "failed to parse api-keys/check response. %s", err)
		return nil, err
	}

	return user, nil
}
