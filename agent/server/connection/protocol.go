package connection

import "github.com/pion/webrtc/v3"

type OutboundMessage struct {
	ConnectionId string           `json:"connection_id,omitempty"`
	DeviceId     string           `json:"device_id,omitempty"`
	Payload      *OutboundPayload `json:"payload,omitempty"`
}

type OutboundPayload struct {
	Type       string                   `json:"type,omitempty"`
	IceServers []webrtc.ICEServer       `json:"ice_servers,omitempty"`
	SDP        string                   `json:"sdp,omitempty"`
	Candidate  *webrtc.ICECandidateInit `json:"candidate,omitempty"`
}

type InboundMessage struct {
	MessageType  string             `json:"message_type,omitempty"`
	IceServers   []webrtc.ICEServer `json:"ice_servers,omitempty"`
	ConnectionId string             `json:"connection_id,omitempty"`
	Payload      *InboundPayload    `json:"payload,omitempty"`
}

type InboundPayload struct {
	Type       string `json:"type,omitempty"`
	SDP        string `json:"sdp,omitempty"`
	Candidate  string `json:"candidate,omitempty"`
	MLineIndex uint16 `json:"mLineIndex,omitempty"`
	MID        string `json:"mid,omitempty"`
}

type InfraConfig struct {
	IceServers []webrtc.ICEServer
}
