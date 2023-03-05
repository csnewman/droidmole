package server

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/emulator"
	"github.com/csnewman/droidmole/agent/server/shell"
	"github.com/csnewman/droidmole/agent/server/sync"
	"github.com/golang/protobuf/ptypes/empty"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type agentControllerServer struct {
	protocol.UnimplementedAgentControllerServer
	log    *zap.SugaredLogger
	server *Server
}

func (s *agentControllerServer) StreamState(e *empty.Empty, server protocol.AgentController_StreamStateServer) error {
	listener := s.server.stateBroadcaster.Listener()

	for {
		state, err := listener.Wait()
		if err != nil {
			s.log.Debug("stopping state stream", err)
			return nil
		}

		err = server.Send(state)
		if err != nil {
			return err
		}
	}
}

func (s *agentControllerServer) StartEmulator(_ context.Context, request *protocol.StartEmulatorRequest) (*empty.Empty, error) {
	s.server.mu.Lock()
	defer s.server.mu.Unlock()

	if s.server.state != StateStopped && s.server.state != StateError {
		err := status.Errorf(codes.FailedPrecondition, "emulator already running")

		if err != nil {
			return nil, err
		}
	} else {
		emu, err := emulator.Start(s.server.adb, request, s.server)

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

	return &empty.Empty{}, nil
}

func (s *agentControllerServer) StopEmulator(_ context.Context, request *protocol.StopEmulatorRequest) (*empty.Empty, error) {
	s.server.mu.Lock()
	defer s.server.mu.Unlock()

	if s.server.state != StateRunning {
		err := status.Errorf(codes.FailedPrecondition, "emulator is not running")

		if err != nil {
			return nil, err
		}
	} else {
		err := s.server.emu.Stop(request)

		if err != nil {
			s.server.state = StateError
			s.server.emu = nil
			s.server.stateError = err
		}
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

	return shell.Process(s.server.adb, server)
}

func (s *agentControllerServer) ListDirectory(ctx context.Context, request *protocol.ListDirectoryRequest) (*protocol.ListDirectoryResponse, error) {
	var respError error

	s.server.mu.Lock()
	if s.server.state != StateRunning {
		respError = status.Errorf(codes.FailedPrecondition, "emulator not running")
	}
	s.server.mu.Unlock()

	if respError != nil {
		return nil, respError
	}

	return sync.ListDirectory(s.server.adb, *request)
}

func (s *agentControllerServer) StatFile(ctx context.Context, request *protocol.StatFileRequest) (*protocol.StatFileResponse, error) {
	var respError error

	s.server.mu.Lock()
	if s.server.state != StateRunning {
		respError = status.Errorf(codes.FailedPrecondition, "emulator not running")
	}
	s.server.mu.Unlock()

	if respError != nil {
		return nil, respError
	}

	return sync.StatFile(s.server.adb, *request)
}

func (s *agentControllerServer) PullFile(request *protocol.PullFileRequest, server protocol.AgentController_PullFileServer) error {
	var respError error

	s.server.mu.Lock()
	if s.server.state != StateRunning {
		respError = status.Errorf(codes.FailedPrecondition, "emulator not running")
	}
	s.server.mu.Unlock()

	if respError != nil {
		return respError
	}

	return sync.PullFile(s.server.adb, *request, server)
}

func (s *agentControllerServer) PushFile(server protocol.AgentController_PushFileServer) error {
	var respError error

	s.server.mu.Lock()
	if s.server.state != StateRunning {
		respError = status.Errorf(codes.FailedPrecondition, "emulator not running")
	}
	s.server.mu.Unlock()

	if respError != nil {
		return respError
	}

	return sync.PushFile(s.server.adb, server)
}
