package server

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/emulator"
	emuproto "github.com/csnewman/droidmole/agent/server/emulator/controller/protocol"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"time"
)

type agentControllerServer struct {
	protocol.UnimplementedAgentControllerServer
	server *Server
}

func (s *agentControllerServer) StreamState(e *empty.Empty, server protocol.AgentController_StreamStateServer) error {
	listener := s.server.stateBroadcaster.Listener()

	for {
		state, err := listener.Wait()
		if err != nil {
			log.Println("stopping state stream", err)
			return nil
		}

		err = server.Send(state)
		if err != nil {
			return err
		}
	}
}

func (s *agentControllerServer) StartEmulator(ctx context.Context, request *protocol.StartEmulatorRequest) (*empty.Empty, error) {
	var respError error

	s.server.mu.Lock()

	if s.server.state != StateStopped {
		respError = status.Errorf(codes.FailedPrecondition, "emulator already running")
	} else {
		emu, err := emulator.Start(request, s.server)

		if err != nil {
			s.server.state = StateError
			s.server.emu = nil
			s.server.stateError = err
		} else {
			s.server.state = StateStarting
			s.server.emu = emu
			s.server.stateError = nil

			s.server.broadcastState()
		}
	}

	s.server.mu.Unlock()

	if respError != nil {
		return nil, respError
	}

	return &empty.Empty{}, nil
}

func (s *agentControllerServer) SendInput(ctx context.Context, event *protocol.TouchEvent) (*empty.Empty, error) {
	touches := make([]*emuproto.Touch, 0)

	for _, e := range event.Touches {
		touches = append(touches, &emuproto.Touch{
			X:          e.X,
			Y:          e.Y,
			Identifier: e.Identifier,
			Pressure:   e.Pressure,
			TouchMajor: e.TouchMajor,
			TouchMinor: e.TouchMinor,
			Expiration: 1,
		})
	}

	return &empty.Empty{}, s.server.conn.SendTouch(emuproto.TouchEvent{
		Touches: touches,
		Display: 0,
	})
}

func (s *agentControllerServer) StreamDisplay(empty *empty.Empty, sds protocol.AgentController_StreamDisplayServer) error {
	dc := &DisplayClient{
		sds: sds,
	}

	s.server.mu.Lock()
	s.server.clients = append(s.server.clients, dc)
	s.server.mu.Unlock()

	// TODO: Remove
	for {
		time.Sleep(time.Second)
	}

	return nil
}

func (s *Server) processSample(sample []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.clients {
		c.ProcessSample(sample)
	}
}

type DisplayClient struct {
	sds          protocol.AgentController_StreamDisplayServer
	videoStarted bool
}

func (c *DisplayClient) ProcessSample(sample []byte) {
	videoKeyframe := (sample[0]&0x1 == 0)
	if videoKeyframe {
		c.videoStarted = true
		//raw := uint(sample.Data[6]) | uint(sample.Data[7])<<8 | uint(sample.Data[8])<<16 | uint(sample.Data[9])<<24
		//width := int(raw & 0x3FFF)
		//height := int((raw >> 16) & 0x3FFF)
	}

	if c.videoStarted {
		err := c.sds.Send(&protocol.DisplayFrame{
			Keyframe: videoKeyframe,
			Data:     sample,
		})
		if err != nil {
			log.Println(err)
		}
	}
}