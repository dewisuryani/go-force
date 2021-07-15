package force

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"

	"github.com/dewisuryani/go-force/forcejson"
)

// Get issues a GET to the specified path with the given params and put the
// umarshalled (json) result in the third parameter
func (forceApi *ForceApi) Get(path string, params url.Values, out interface{}) error {
	return forceApi.request("GET", path, params, nil, out)
}

// Post issues a POST to the specified path with the given params and payload
// and put the unmarshalled (json) result in the third parameter
func (forceApi *ForceApi) Post(path string, params url.Values, payload, out interface{}) error {
	return forceApi.request("POST", path, params, payload, out)
}

// Put issues a PUT to the specified path with the given params and payload
// and put the unmarshalled (json) result in the third parameter
func (forceApi *ForceApi) Put(path string, params url.Values, payload, out interface{}) error {
	return forceApi.request("PUT", path, params, payload, out)
}

// Patch issues a PATCH to the specified path with the given params and payload
// and put the unmarshalled (json) result in the third parameter
func (forceApi *ForceApi) Patch(path string, params url.Values, payload, out interface{}) error {
	return forceApi.request("PATCH", path, params, payload, out)
}

// Delete issues a DELETE to the specified path with the given payload
func (forceApi *ForceApi) Delete(path string, params url.Values) error {
	return forceApi.request("DELETE", path, params, nil, nil)
}

func (forceApi *ForceApi) request(method, path string, params url.Values, payload, out interface{}) error {
	if err := forceApi.oauth.Validate(); err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"method": method,
			"err":    err,
		}).Error("error creating request")
		return err
	}

	// Build Uri
	var uri bytes.Buffer
	uri.WriteString(forceApi.oauth.InstanceUrl)
	uri.WriteString(path)
	if params != nil && len(params) != 0 {
		uri.WriteString("?")
		uri.WriteString(params.Encode())
	}

	// Build body
	var body io.Reader
	if payload != nil {

		jsonBytes, err := forcejson.Marshal(payload)
		if err != nil {
			err = tracerr.Wrap(err)
			logrus.WithFields(logrus.Fields{
				"payload": payload,
				"err":     err,
			}).Error("error marshaling encoded payload")
			return err
		}

		body = bytes.NewReader(jsonBytes)
	}

	// Build Request
	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"method": method,
			"uri":    uri.String(),
			"body":   body,
			"err":    err,
		}).Error("error creating http new request")
		return err
	}

	// Add Headers
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", jsonType)
	req.Header.Set("Accept", jsonType)
	req.Header.Set("Authorization", fmt.Sprintf("%v %v", "Bearer", forceApi.oauth.AccessToken))

	// Send
	forceApi.traceRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"method": method,
			"req":    req,
			"err":    err,
		}).Error("error client do")
		return err
	}
	defer resp.Body.Close()
	forceApi.traceResponse(resp)

	// Sometimes the force API returns no body, we should catch this early
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = tracerr.Wrap(err)
		logrus.WithFields(logrus.Fields{
			"body": resp.Body,
			"err":  err,
		}).Error("error reading response bytes")
		return err
	}
	forceApi.traceResponseBody(respBytes)

	// Attempt to parse response into out
	var objectUnmarshalErr error
	if out != nil {
		objectUnmarshalErr = forcejson.Unmarshal(respBytes, out)
		if objectUnmarshalErr == nil {
			return nil
		}
	}

	// Attempt to parse response as a force.com api error before returning object unmarshal err
	apiErrors := APIErrors{}
	if marshalErr := forcejson.Unmarshal(respBytes, &apiErrors); marshalErr == nil {
		if apiErrors.Validate() {
			// Check if error is oauth token expired
			if forceApi.oauth.Expired(apiErrors) {
				// Reauthenticate then attempt query again
				oauthErr := forceApi.oauth.Authenticate()
				if oauthErr != nil {
					return oauthErr
				}

				return forceApi.request(method, path, params, payload, out)
			}

			return apiErrors
		}
	}

	if objectUnmarshalErr != nil {
		// Not a force.com api error. Just an unmarshalling error.
		err = tracerr.Wrap(objectUnmarshalErr)
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("error unmarshal response to object")
		return err
	}

	// Sometimes no response is expected. For example delete and update. We still have to make sure an error wasn't returned.
	return nil
}

func (forceApi *ForceApi) traceRequest(req *http.Request) {
	if forceApi.logger != nil {
		forceApi.trace("Request:", req, "%v")
	}
}

func (forceApi *ForceApi) traceResponse(resp *http.Response) {
	if forceApi.logger != nil {
		forceApi.trace("Response:", resp, "%v")
	}
}

func (forceApi *ForceApi) traceResponseBody(body []byte) {
	if forceApi.logger != nil {
		forceApi.trace("Response Body:", string(body), "%s")
	}
}
