package server

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/emulator"
	"github.com/csnewman/droidmole/agent/server/shell"
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

func (s *agentControllerServer) SendInput(_ context.Context, request *protocol.InputRequest) (*empty.Empty, error) {
	s.server.mu.Lock()
	defer s.server.mu.Unlock()

	if request == nil {
		return nil, status.Errorf(codes.InvalidArgument, "no request given")
	}

	if s.server.state == StateRunning || s.server.state == StateStarting {
		return &empty.Empty{}, s.server.emu.ProcessInput(*request)
	}

	return nil, status.Errorf(codes.FailedPrecondition, "emulator not running")
}

func (s *agentControllerServer) StreamSysLog(_ *empty.Empty, server protocol.AgentController_StreamSysLogServer) error {
	listener := s.server.syslog.Listen()

	for {
		line := listener.Recv()

		err := server.Send(&protocol.SysLogEntry{
			Line: line,
		})
		if err != nil {
			return err
		}
	}
}

func (s *agentControllerServer) OpenShell(server protocol.AgentController_OpenShellServer) error {
	var respError error

	s.server.mu.Lock()
	if s.server.state != StateRunning {
		respError = status.Errorf(codes.FailedPrecondition, "emulator not running")
	}
	s.server.mu.Unlock()

	if respError != nil {
		return respError
	}

	return shell.Process(server)
}
