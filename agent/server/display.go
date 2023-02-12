package server

import (
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/emulator"
	"github.com/csnewman/droidmole/agent/util/broadcaster"
	"github.com/csnewman/droidmole/agent/util/vpx"
	"github.com/golang/protobuf/ptypes/empty"
	"log"
	"time"
)

func (s *agentControllerServer) StreamDisplay(_ *empty.Empty, sds protocol.AgentController_StreamDisplayServer) error {
	frameListener := s.server.frameBroadcaster.Listener()

	ticker := time.NewTicker(1 * time.Second / 5)
	// TODO: Implement
	done := make(chan bool)

	dp := &displayProcessor{
		sds:           sds,
		frameListener: frameListener,
	}

	err := dp.processFrame()
	if err != nil {
		log.Println("Error streaming display: ", err)
		return err
	}

	for {
		select {
		case <-done:
			return nil
		case <-ticker.C:
			err := dp.processFrame()
			if err != nil {
				log.Println("Error streaming display: ", err)
				return err
			}
		}
	}
}

type displayProcessor struct {
	sds           protocol.AgentController_StreamDisplayServer
	frameListener *broadcaster.Listener[*emulator.Frame]
	img           *vpx.Image
	codecCtx      *vpx.CodecCtx
	width         int
	height        int
	frameCount    int
}

func (p *displayProcessor) processFrame() error {
	frame, err := p.frameListener.Wait()
	if err != nil {
		return err
	}

	if frame == nil {
		p.width = 0
		p.height = 0
		p.frameCount = 0
		log.Println("Changing stream resolution ", p.width, "x", p.height)

		if p.img != nil {
			p.img.Free()
		}

		if p.codecCtx != nil {
			p.codecCtx.Free()
		}

		return p.sds.Send(&protocol.DisplayFrame{
			Keyframe: true,
			Width:    int32(0),
			Height:   int32(0),
			Data:     []byte{},
		})
	}

	if frame.Width != p.width || frame.Height != p.height {
		p.width = frame.Width
		p.height = frame.Height
		p.frameCount = 0
		log.Println("Changing stream resolution ", p.width, "x", p.height)

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
			log.Panic("failed to create img")
		}
	}

	data := vpx.RgbToYuv(frame.Data, 720, 1280)
	p.img.Read(data)

	keyframe := p.frameCount%20 == 0

	flags := vpx.EFlagNone
	if keyframe {
		flags = vpx.EFlagForceKF
	}

	err = p.codecCtx.Encode(p.img, vpx.CodecPts(p.frameCount), uint64(1), flags, vpx.DLRealtime)
	if err != nil {
		log.Fatal("scr error", err)
	}

	var iter vpx.CodecIter

	for {
		pkt := p.codecCtx.GetFrameBuffer(&iter)
		if pkt == nil {
			break
		}

		err := p.sds.Send(&protocol.DisplayFrame{
			Keyframe: keyframe,
			Width:    int32(p.width),
			Height:   int32(p.height),
			Data:     pkt,
		})
		if err != nil {
			return err
		}
	}

	p.frameCount++

	return nil

}
