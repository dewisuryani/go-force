package force

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

type forceOauth struct {
	AccessToken string `json:"access_token"`
	InstanceUrl string `json:"instance_url"`
	Id          string `json:"id"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`

	loginURI      string
	clientId      string
	clientSecret  string
	refreshToken  string
	userName      string
	password      string
	securityToken string
	environment   string
}

func (oauth *forceOauth) Validate() error {
	if oauth == nil || len(oauth.InstanceUrl) == 0 || len(oauth.AccessToken) == 0 {
		return fmt.Errorf("Invalid Force Oauth Object: %#v", oauth)
	}

	return nil
}

func (oauth *forceOauth) Expired(apiErrors APIErrors) bool {
	for _, err := range apiErrors {
		if err.ErrorCode == invalidSessionErrorCode {
			return true
		}
	}

	return false
}

func (oauth *forceOauth) Authenticate() error {
	payload := url.Values{
		"grant_type":    {grantType},
		"client_id":     {oauth.clientId},
		"client_secret": {oauth.clientSecret},
		"username":      {oauth.userName},
		"password":      {fmt.Sprintf("%v%v", oauth.password, oauth.securityToken)},
	}

	return oauth.AuthenticateWithPayload(payload)
}

func (oauth *forceOauth) AuthenticateWithRefreshToken() error {
	payload := url.Values{
		"grant_type":    {grantTypeRefreshToken},
		"client_id":     {oauth.clientId},
		"client_secret": {oauth.clientSecret},
		"refresh_token": {oauth.refreshToken},
	}

	return oauth.AuthenticateWithPayload(payload)
}

func (oauth *forceOauth) AuthenticateWithPayload(payload url.Values) error {
	// Build Uri
	uri := oauth.loginURI + oauthURL

	// Build Body
	body := strings.NewReader(payload.Encode())

	// Build Request
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"uri":  uri,
			"body": body,
			"err":  err,
		}).Error("error creating http new request")
		return err
	}

	// Add Headers
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", jsonType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"req": req,
			"err": err,
		}).Error("error client do on authenticate with payload")
		return err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"body": resp.Body,
			"err":  err,
		}).Error("error reading authentication response bytes")
		return err
	}

	// Attempt to parse response as a force.com api error
	apiError := &APIError{}
	if err := json.Unmarshal(respBytes, apiError); err == nil {
		// Check if api error is valid
		if apiError.Validate() {
			return apiError
		}
	}

	if err := json.Unmarshal(respBytes, oauth); err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"oauth":     oauth,
			"respBytes": string(respBytes),
			"err":       err,
		}).Error("error unmarshal authentication response")
		return err
	}

	return nil
}
