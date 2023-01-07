package connection

import (
	"errors"
	"fmt"
	"github.com/pion/webrtc/v3"
	"log"
)

type CfConnection struct {
	connector *PollingConnector
	rtc       *webrtc.PeerConnection
}

func New(serverUrl string) (*CfConnection, error) {
	con, err := NewConnector(serverUrl)
	if err != nil {
		log.Fatal(err)
	}

	devices, err := con.Devices()
	if err != nil {
		return nil, err
	}

	if len(devices) != 1 {
		return nil, fmt.Errorf("unexpected number of devices: %d", len(devices))
	}

	config, err := con.GetConfig()
	if err != nil {
		return nil, err
	}

	rtc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: config.IceServers,
	})
	if err != nil {
		return nil, err
	}

	rtc.OnNegotiationNeeded(func() {
		log.Println("negotiation needed")
	})

	err = con.Connect(devices[0])
	if err != nil {
		return nil, err
	}

	cfc := &CfConnection{
		connector: con,
		rtc:       rtc,
	}

	cfc.connector.ConfigureCallback(cfc.processMessage)
	rtc.OnICECandidate(cfc.processICECandidate)

	rtc.OnDataChannel(func(channel *webrtc.DataChannel) {
		log.Println("data", channel)
	})

	rtc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("state", state)
	})

	rtc.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("id", remote.ID())
		log.Println("stream-id", remote.StreamID())
		log.Println("payload-type", remote.PayloadType())
		log.Println("kind", remote.Kind())
		log.Println("codec", remote.Codec())
		log.Println(" type", remote.Codec().PayloadType)
		log.Println(" mime", remote.Codec().MimeType)
		log.Println(" channels", remote.Codec().Channels)
		log.Println(" clock-rate", remote.Codec().ClockRate)
		log.Println(" feedback", remote.Codec().RTCPFeedback)
		log.Println(" fmtp-line", remote.Codec().SDPFmtpLine)
		log.Println(" capability", remote.Codec().RTPCodecCapability)
	})

	err = con.RequestOffer(config.IceServers)
	if err != nil {
		return nil, err
	}

	return cfc, nil
}

func (c *CfConnection) processMessage(msg *InboundPayload) {
	err := c.processMessageInner(msg)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *CfConnection) processMessageInner(msg *InboundPayload) error {
	switch msg.Type {
	case "offer":
		err := c.rtc.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  msg.SDP,
		})
		if err != nil {
			return err
		}

		answer, err := c.rtc.CreateAnswer(nil)
		if err != nil {
			return err
		}

		if answer.Type != webrtc.SDPTypeAnswer {
			return errors.New("unexpected answer type")
		}

		err = c.connector.SendAnswer(answer.SDP)
		if err != nil {
			return err
		}

		err = c.rtc.SetLocalDescription(answer)
		if err != nil {
			return err
		}

		return nil
	case "ice-candidate":
		err := c.rtc.AddICECandidate(webrtc.ICECandidateInit{
			Candidate:        msg.Candidate,
			SDPMid:           &msg.MID,
			SDPMLineIndex:    &msg.MLineIndex,
			UsernameFragment: nil,
		})
		if err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (c *CfConnection) processICECandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		log.Println("Ignoring null candidate")
		return
	}
	log.Println(candidate)

	err := c.connector.SendCandidate(candidate.ToJSON())
	if err != nil {
		log.Fatal(err)
	}
}
