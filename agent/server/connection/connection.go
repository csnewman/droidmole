package connection

import (
	"bytes"
	media2 "droidmole/server/media"
	"errors"
	"fmt"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/xlab/libvpx-go/vpx"
	"image/jpeg"
	"os"
	"sync"

	"io"
	"log"
	"time"
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

	m := &webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil,
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))
	rtc, err := api.NewPeerConnection(webrtc.Configuration{
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

	rtc.OnTrack(cfc.processTrack)

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

func (c *CfConnection) processTrack(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
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

	log.Printf("Track has started, of type %d: %s \n", remote.PayloadType(), remote.Codec().RTPCodecCapability.MimeType)

	go func() {
		ticker := time.NewTicker(time.Millisecond * 250)
		for range ticker.C {
			errSend := c.rtc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(remote.SSRC())}})
			if errSend != nil {
				fmt.Println(errSend)
			}
		}
	}()

	switch remote.Kind() {
	//case webrtc.RTPCodecTypeAudio:
	//	saver.PushOpus(rtp)
	case webrtc.RTPCodecTypeVideo:
		c.processVideo(remote)
	}
}

type VideoProcessor struct {
	mu        sync.Mutex
	hasKf     bool
	samplesId uint64
	samples   []*media.Sample
}

func (v *VideoProcessor) generateScreenshots() {
	iface := vpx.DecoderIfaceVP8()
	ctx := vpx.NewCodecCtx()

	err := vpx.Error(vpx.CodecDecInitVer(ctx, iface, nil, 0, vpx.DecoderABIVersion))
	if err != nil {
		log.Fatal("codec init", err)
	}

	var lastSamplesId uint64
	samplesPos := -1

	ticker := time.NewTicker(time.Millisecond * 3000)
	for range ticker.C {
		var samples []*media.Sample

		v.mu.Lock()
		if !v.hasKf {
			v.mu.Unlock()
			continue
		}

		if lastSamplesId != v.samplesId {
			samplesPos = -1
			lastSamplesId = v.samplesId
		}

		samples = v.samples

		v.mu.Unlock()

		var lastImage *vpx.Image

		for i, sample := range samples {
			if i <= samplesPos {
				continue
			}
			samplesPos = i

			dataSize := uint32(len(sample.Data))
			err = vpx.Error(vpx.CodecDecode(ctx, string(sample.Data), dataSize, nil, 0))
			if err != nil {
				log.Fatal("decode error", err)
			}

			var iter vpx.CodecIter
			img := vpx.CodecGetFrame(ctx, &iter)
			if img != nil {
				if lastImage != nil {
					lastImage.Free()
				}

				lastImage = img
			}
		}

		if lastImage == nil {
			log.Println("No new frames available")
			continue
		}

		lastImage.Deref()

		buffer := new(bytes.Buffer)
		if err = jpeg.Encode(buffer, lastImage.ImageYCbCr(), nil); err != nil {
			log.Println("jpeg Encode Error", err)
			continue
		}

		err = os.WriteFile("frame.jpeg", buffer.Bytes(), 0644)
		if err != nil {
			panic(err)
		}

		lastImage.Free()

		log.Println("Generated new frame")
	}
}

func (v *VideoProcessor) Push(sample *media.Sample) {
	v.mu.Lock()
	defer v.mu.Unlock()

	videoKeyframe := sample.Data[0]&0x1 == 0

	if videoKeyframe {
		v.samples = []*media.Sample{
			sample,
		}
		v.hasKf = true
		v.samplesId++
	} else if !v.hasKf {
		return
	} else {
		v.samples = append(v.samples, sample)
	}
}

func (c *CfConnection) processVideo(remote *webrtc.TrackRemote) {

	vp := &VideoProcessor{
		mu: sync.Mutex{},
	}

	go vp.generateScreenshots()

	vs := media2.NewVideoSampler()
	for {
		rtp, _, readErr := remote.ReadRTP()
		if readErr != nil {
			if readErr == io.EOF {
				log.Println("stream EOF")
				return
			}
			panic(readErr)
		}

		vs.Push(rtp)

		sample := vs.Pop()
		if sample == nil {
			continue
		}

		vp.Push(sample)
	}
}
