package server

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/emulator"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
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
	s.server.mu.Lock()

	var respError error
	if s.server.state == StateRunning || s.server.state == StateStarting {
		respError = s.server.emu.ProcessInput(event)
	} else {
		respError = status.Errorf(codes.FailedPrecondition, "emulator not running")
	}

	s.server.mu.Unlock()

	if respError != nil {
		return nil, respError
	}

	return &empty.Empty{}, nil
}
