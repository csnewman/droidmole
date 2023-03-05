package server

import (
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/emulator"
	"github.com/csnewman/droidmole/agent/util/broadcaster"
	"github.com/csnewman/droidmole/agent/util/vpx"
	"go.uber.org/zap"
	"time"
)

func (s *agentControllerServer) StreamDisplay(request *protocol.StreamDisplayRequest, sds protocol.AgentController_StreamDisplayServer) error {
	frameListener := s.server.frameBroadcaster.Listener()

	dp := &displayProcessor{
		log:           s.log,
		sds:           sds,
		frameListener: frameListener,
		request:       request,
	}

	err := dp.processFrame()
	if err != nil {
		s.log.Error("Error streaming display: ", err)
		return err
	}

	if request.MaxFps == 0 {
		for {
			err := dp.processFrame()
			if err != nil {
				s.log.Error("Error streaming display: ", err)
				return err
			}

		}
	} else {
		ticker := time.NewTicker(1 * time.Second / time.Duration(request.MaxFps))
		// TODO: Implement
		done := make(chan bool)

		for {
			select {
			case <-done:
				return nil
			case <-ticker.C:
				err := dp.processFrame()
				if err != nil {
					s.log.Error("Error streaming display: ", err)
					return err
				}
			}
		}
	}
}

type displayProcessor struct {
	log           *zap.SugaredLogger
	sds           protocol.AgentController_StreamDisplayServer
	frameListener *broadcaster.Listener[*emulator.Frame]
	img           *vpx.Image
	codecCtx      *vpx.CodecCtx
	request       *protocol.StreamDisplayRequest
	width         uint32
	height        uint32
	frameCount    uint32
	lastKeyframe  time.Time
}

func (p *displayProcessor) processFrame() error {
	frame, err := p.frameListener.Wait()
	if err != nil {
		return err
	}

	now := time.Now()

	// Blank screen
	if frame == nil {
		p.width = 0
		p.height = 0
		p.frameCount = 0
		p.lastKeyframe = now
		p.log.Info("Changing stream resolution ", p.width, "x", p.height)

		if p.img != nil {
			p.img.Free()
		}

		if p.codecCtx != nil {
			p.codecCtx.Free()
		}

		return p.sds.Send(&protocol.DisplayFrame{
			Keyframe: true,
			Width:    uint32(0),
			Height:   uint32(0),
			Data:     []byte{},
		})
	}

	// Detect display size change
	if frame.Width != p.width || frame.Height != p.height {
		p.width = frame.Width
		p.height = frame.Height
		p.frameCount = 0
		p.lastKeyframe = now
		p.log.Info("Changing stream resolution ", p.width, "x", p.height)

		// Reconfigure encoder
		if p.img != nil {
			p.img.Free()
		}

		if p.codecCtx != nil {
			p.codecCtx.Free()
		}

		// TODO: Check memory freeing
		vp8 := vpx.VP8Iface()

		encCfg := vpx.NewCodecEncCfg()
		err = encCfg.Default(vp8)
		if err != nil {
			return err
		}

		encCfg.SetGW(uint(p.width))
		encCfg.SetGH(uint(p.height))
		encCfg.SetRcTargetBitrate(1_000)
		encCfg.SetGErrorResilient(1)
		encCfg.SetGTimebase(1, 60)

		p.codecCtx = vpx.NewCodecCtx()
		err = p.codecCtx.EncInit(vp8, encCfg, 0)
		if err != nil {
			return err
		}

		p.img = vpx.NullImage().Alloc(vpx.ImageFormatI420, uint32(p.width), uint32(p.height), 0)
		if p.img == nil {
			p.log.Fatal("failed to create img")
		}
	}

	// Convert frame to YUV
	data := vpx.RgbToYuv(frame.Data, p.width, p.height)
	p.img.Read(data)

	// Determine whether to encode a keyframe
	keyframe := p.frameCount == 0

	if p.request.KeyframeInterval != 0 && uint32(now.Sub(p.lastKeyframe).Milliseconds()) >= p.request.KeyframeInterval {
		keyframe = true
		p.lastKeyframe = now
	}

	flags := vpx.EFlagNone
	if keyframe {
		flags = vpx.EFlagForceKF
	}

	// Encode
	err = p.codecCtx.Encode(p.img, vpx.CodecPts(p.frameCount), uint64(1), flags, vpx.DLRealtime)
	if err != nil {
		p.log.Fatal("scr error", err)
	}

	// Extract packets
	var iter vpx.CodecIter
	for {
		pkt := p.codecCtx.GetFrameBuffer(&iter)
		if pkt == nil {
			break
		}

		err := p.sds.Send(&protocol.DisplayFrame{
			Keyframe: keyframe,
			Width:    p.width,
			Height:   p.height,
			Data:     pkt,
		})
		if err != nil {
			return err
		}
	}

	p.frameCount++

	return nil
}
