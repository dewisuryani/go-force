package force

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/publicsuffix"
)

//StreamsForce struct
type StreamsForce struct {
	APIForce       *ForceApi
	ClientID       string
	Subscribes     map[string]func([]byte, ...interface{})
	Timeout        int
	LongPoolClient *http.Client
}

//CometdVersion global var
var (
	CometdVersion string = "40.0"
	TopicMode            = map[string]string{
		"CDC":       "/data/%vChangeEvent",
		"Event":     "/event/%v",
		"PushTopic": "/topic/%v",
	}
)

func (s *StreamsForce) httpPost(payload string) (*http.Response, error) {
	ioPayload := strings.NewReader(payload)
	endpoint := s.APIForce.oauth.InstanceUrl + "/cometd/" + CometdVersion
	headerVal := "OAuth " + s.APIForce.oauth.AccessToken

	request, _ := http.NewRequest("POST", endpoint, ioPayload)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", headerVal)
	logrus.WithFields(logrus.Fields{
		"payload":  ioPayload,
		"endpoint": endpoint,
		"header":   headerVal,
		"request":  request,
	}).Info("http post stream")

	resp, err := s.LongPoolClient.Do(request)
	logrus.WithFields(logrus.Fields{
		"request":  request,
		"response": resp,
		"err":      err,
	}).Info("long pool client do")

	return resp, err
}

func (s *StreamsForce) performTask(params string) ([]byte, error) {
	resp, err := s.httpPost(params)
	if err != nil {
		log.Fatal(err)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"param":    params,
		"resp":     resp,
		"respBody": string(respBytes),
		"err":      err,
	}).Info("perform task")

	return respBytes, err
}

func (s *StreamsForce) connect() ([]byte, error) {
	connectParams := `{ "channel": "/meta/connect", "clientId": "` + s.ClientID + `", "connectionType": "long-polling"}`
	return s.performTask(connectParams)
}

func (s *StreamsForce) handshake() ([]byte, error) {
	handshakeParams := `{"channel":"/meta/handshake", "supportedConnectionTypes":["long-polling"], "version":"1.0"}`
	return s.performTask(handshakeParams)
}

func (s *StreamsForce) disconnect() ([]byte, error) {
	connectParams := `{ "channel": "/meta/disconnect", "clientId": "` + s.ClientID + `", "connectionType": "long-polling"}`
	return s.performTask(connectParams)
}

//ConnectToStreamingAPI connects to streaming API
func (forceAPI *ForceApi) ConnectToStreamingAPI() {
	//set up the client
	cookiejarOptions := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&cookiejarOptions)
	if err != nil {
		log.Fatal(err)
	}
	forceAPI.stream = &StreamsForce{
		APIForce:       forceAPI,
		ClientID:       "",
		Subscribes:     map[string]func([]byte, ...interface{}){},
		Timeout:        0,
		LongPoolClient: &http.Client{Jar: jar},
	}

	//handshake
	handshakeBytes, err := forceAPI.stream.handshake()
	if err != nil {
		log.Fatal("Handshake failed! Err = ", err)
	}
	var data []map[string]interface{}
	json.Unmarshal(handshakeBytes, &data)

	//DEBUG ONLY (DELETE LATER)
	fmt.Println(data)

	forceAPI.stream.ClientID = data[0][clientIDParam].(string)

	//connect
	connBytes, err := forceAPI.stream.connect()
	if err != nil {
		log.Fatal("Connect failed! Err = ", err)
	}
	var connectData []map[string]interface{}
	json.Unmarshal(connBytes, &connectData)
	for _, msg := range data {
		cb := forceAPI.stream.Subscribes[msg[channelParam].(string)]
		if cb != nil {
			cb(connBytes)
		}
	}

	//DEBUG ONLY (DELETE LATER)
	fmt.Println(string(connBytes))

	go func() {
		for {
			connBytes, err = forceAPI.stream.connect()
			if err != nil {
				log.Fatal("Connect failed! Err = ", err)
			}
			json.Unmarshal(connBytes, &connectData)
			for _, msg := range connectData {
				cb := forceAPI.stream.Subscribes[msg[channelParam].(string)]
				if cb != nil {
					cb(connBytes)
				}
			}
		}
	}()

}

func (forceAPI *ForceApi) DisconnectStreamingAPI() {
	connByte, err := forceAPI.stream.disconnect()
	if err != nil {
		log.Fatal("Disconnect failed! Err = ", err)
	}

	logrus.WithFields(logrus.Fields{
		"response": string(connByte),
		"err":      err,
	}).Info("disconnect streaming api")
}

func getTopic(mode, topic string) string {
	topicMode, ok := TopicMode[mode]
	if !ok {
		log.Fatal("Invalid mode!")
	}
	return fmt.Sprintf(topicMode, topic)
}

//Subscribe receives message from any mode such as:
// "CDC" : Change Data Capture
// "PushTopic" : Push Topic
// "Event" : Event
func (forceAPI *ForceApi) Subscribe(mode, topic string, callback func([]byte, ...interface{})) ([]byte, error) {
	//Get topic by mode
	topicString := getTopic(mode, topic)
	subscribeParams := `{ "channel": "/meta/subscribe", "clientId": "` + forceAPI.stream.ClientID + `", "subscription": "` + topicString + `"}`

	subscribeBytes, err := forceAPI.stream.performTask(subscribeParams)
	if err != nil {
		log.Fatal(err)
	}
	forceAPI.stream.Subscribes[topicString] = callback

	return subscribeBytes, err
}

func (forceAPI *ForceApi) Unsubscribe(mode, topic string) error {
	topicString := getTopic(mode, topic)
	if _, ok := forceAPI.stream.Subscribes[topicString]; !ok {
		return errors.New("this topic hasn't been subscribed")
	}

	unsubscribeParams := `{ "channel": "/meta/unsubscribe", "clientId": "` + forceAPI.stream.ClientID + `", "subscription": "` + topicString + `"}`
	_, err := forceAPI.stream.performTask(unsubscribeParams)
	if err != nil {
		log.Fatal(err)
	}

	delete(forceAPI.stream.Subscribes, topicString)
	return nil
}

//Unsubscribe still doesn't do anything yet
func Unsubscribe(topic string) {
	fmt.Println(topic)
}

//DisconnectStreamingAPI still doesn't do anything yet
func DisconnectStreamingAPI() {
}
