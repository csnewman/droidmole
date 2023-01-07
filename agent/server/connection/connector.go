package connection

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pion/webrtc/v3"
	"log"
	"net/http"
	"net/url"
	"time"
)

type MessageCallback func(msg *InboundPayload)

type PollingConnector struct {
	baseURL      *url.URL
	httpClient   *http.Client
	newEndpoints bool
	connectionId string
	callback     *MessageCallback
	messageCount int
}

func NewConnector(serverUrl string) (*PollingConnector, error) {
	parsed, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &PollingConnector{
		baseURL: parsed,
		httpClient: &http.Client{
			Transport: customTransport,
		},
	}, nil
}

func (c *PollingConnector) Devices() ([]string, error) {
	rel := &url.URL{Path: "/devices"}
	u := c.baseURL.ResolveReference(rel)
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	var msg []string
	err = json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (c *PollingConnector) GetConfig() (*InfraConfig, error) {
	rel := &url.URL{Path: "/infra_config"}
	u := c.baseURL.ResolveReference(rel)
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	var msg InboundMessage
	err = json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil {
		return nil, err
	}

	if msg.MessageType != "config" {
		return nil, errors.New("unexpected message type")
	}

	return &InfraConfig{IceServers: msg.IceServers}, nil
}

func (c *PollingConnector) Connect(deviceId string) error {
	rel := &url.URL{Path: "/connect"}
	if c.newEndpoints {
		rel = &url.URL{Path: "/polled_connections"}
	}

	u := c.baseURL.ResolveReference(rel)

	jsonValue, err := json.Marshal(OutboundMessage{
		DeviceId: deviceId,
	})
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(u.String(), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		if c.newEndpoints {
			return errors.New("unexpected protocol version")
		}

		c.newEndpoints = true
		return c.Connect(deviceId)
	}

	var msg InboundMessage
	err = json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil {
		return err
	}

	if msg.ConnectionId == "" {
		return errors.New("unexpected message type")
	}

	c.connectionId = msg.ConnectionId
	return nil
}

func (c *PollingConnector) ConfigureCallback(callback MessageCallback) {
	c.callback = &callback
	go c.callbackProcessor()
}

func (c *PollingConnector) callbackProcessor() {
	delay := 1000

	for {
		delay += delay
		if delay > 10000 {
			delay = 10000
		}

		payloads, err := c.PollMessages()
		if err != nil {
			log.Fatal(err)
		}

		if len(payloads) > 0 {
			delay = 100
		}

		for _, payload := range payloads {
			(*c.callback)(payload)
		}

		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

func (c *PollingConnector) PollMessages() ([]*InboundPayload, error) {
	var resp *http.Response

	if c.newEndpoints {
		rel := &url.URL{
			Path:     fmt.Sprintf("/polled_connections/%s/messages", c.connectionId),
			RawQuery: fmt.Sprintf("start=%d", c.messageCount),
		}
		u := c.baseURL.ResolveReference(rel)

		var err error
		resp, err = c.httpClient.Get(u.String())
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		var inMsg []InboundMessage
		err = json.NewDecoder(resp.Body).Decode(&inMsg)
		if err != nil {
			return nil, err
		}

		c.messageCount += len(inMsg)
		var payloads []*InboundPayload

		for _, message := range inMsg {
			if message.MessageType != "device_msg" {
				return nil, fmt.Errorf("unexpected message type %s", message.MessageType)
			}

			payloads = append(payloads, message.Payload)
		}

		return payloads, nil
	} else {
		rel := &url.URL{Path: "/poll_messages"}
		u := c.baseURL.ResolveReference(rel)
		jsonValue, err := json.Marshal(OutboundMessage{
			ConnectionId: c.connectionId,
		})
		if err != nil {
			return nil, err
		}

		resp, err = c.httpClient.Post(u.String(), "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		panic("Old format response not implemented")
	}
}

func (c *PollingConnector) RequestOffer(iceServers []webrtc.ICEServer) error {
	return c.Forward(OutboundPayload{
		Type:       "request-offer",
		IceServers: iceServers,
	})
}

func (c *PollingConnector) SendAnswer(sdp string) error {
	return c.Forward(OutboundPayload{
		Type: "answer",
		SDP:  sdp,
	})
}

func (c *PollingConnector) SendCandidate(candidate webrtc.ICECandidateInit) error {
	return c.Forward(OutboundPayload{
		Type:      "ice-candidate",
		Candidate: &candidate,
	})
}

func (c *PollingConnector) Forward(payload OutboundPayload) error {
	rel := &url.URL{Path: "/forward"}
	if c.newEndpoints {
		rel = &url.URL{Path: fmt.Sprintf("/polled_connections/%s/:forward", c.connectionId)}
	}

	u := c.baseURL.ResolveReference(rel)

	jsonValue, err := json.Marshal(OutboundMessage{
		ConnectionId: c.connectionId,
		Payload:      &payload,
	})
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(u.String(), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if c.newEndpoints {
		var inMsg string
		err = json.NewDecoder(resp.Body).Decode(&inMsg)
		if err != nil {
			return err
		}

		if inMsg != "ok" {
			return errors.New(inMsg)
		}

		return nil
	} else {
		panic("Old format response not implemented")
	}
}
