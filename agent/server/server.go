package server

import (
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/adb"
	"github.com/csnewman/droidmole/agent/server/emulator"
	"github.com/csnewman/droidmole/agent/server/syslog"
	"github.com/csnewman/droidmole/agent/util/broadcaster"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"sync"
	"time"
)

type State int

const (
	StateError    State = 0
	StateStopped        = 1
	StateStarting       = 2
	StateRunning        = 3
)

type Server struct {
	log *zap.SugaredLogger
	adb adb.Adb

	state            State
	stateError       error
	mu               sync.Mutex
	emu              *emulator.Emulator
	stateBroadcaster *broadcaster.Broadcaster[*protocol.AgentState]
	frameBroadcaster *broadcaster.Broadcaster[*emulator.Frame]
	syslog           *syslog.SysLog
}

func New(log *zap.SugaredLogger, adb adb.Adb) *Server {
	return &Server{
		log:              log,
		adb:              adb,
		state:            StateStopped,
		stateBroadcaster: broadcaster.New[*protocol.AgentState](),
		frameBroadcaster: broadcaster.New[*emulator.Frame](),
	}
}

func (s *Server) Start() {
	s.log.Info("Starting agent server")

	s.broadcastState()

	go s.startHeartbeat()

	err := s.adb.StartServer()
	if err != nil {
		s.log.Fatal("failed to start adb server", err)
	}

	s.syslog, err = syslog.Start()
	if err != nil {
		s.log.Fatal("failed to start syslog", err)
	}

	lis, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		s.log.Fatal("failed to listen", err)
	}

	acs := &agentControllerServer{
		log:    s.log,
		server: s,
	}

	grpcServer := grpc.NewServer()
	protocol.RegisterAgentControllerServer(grpcServer, acs)

	s.log.Info("Servicing requests")
	s.log.Fatal(grpcServer.Serve(lis))
}

func (s *Server) startHeartbeat() {
	ticker := time.NewTicker(1 * time.Second)
	// TODO: Implement
	done := make(chan bool)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			s.heartbeat()
		}
	}
}

func (s *Server) heartbeat() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO: Check state

	s.broadcastState()
}

func (s *Server) broadcastState() {
	newState := &protocol.AgentState{}

	switch s.state {
	case StateError:
		newState.EmulatorState = protocol.AgentState_ERROR
		msg := s.stateError.Error()
		newState.EmulatorError = &msg
	case StateStopped:
		newState.EmulatorState = protocol.AgentState_OFF
	case StateStarting:
		newState.EmulatorState = protocol.AgentState_STARTING
	case StateRunning:
		newState.EmulatorState = protocol.AgentState_RUNNING
	}

	s.stateBroadcaster.Broadcast(newState)
}

func (s *Server) OnEmulatorExit(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.log.Info("Emulator exited with error:", err)
		s.state = StateError
		s.stateError = err
	} else {
		s.log.Info("Emulator cleanly exited")
		s.state = StateStopped
		s.stateError = nil
	}

	s.broadcastState()
}

func (s *Server) OnEmulatorStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = StateRunning

	s.broadcastState()
}

func (s *Server) OnEmulatorFrame(frame emulator.Frame) {
	s.frameBroadcaster.Broadcast(&frame)
}
