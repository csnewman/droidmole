package server

import (
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/adb"
	"github.com/csnewman/droidmole/agent/server/controller"
	"github.com/csnewman/droidmole/agent/server/emulator"
	"github.com/csnewman/droidmole/agent/util/broadcaster"
	"google.golang.org/grpc"
	"log"
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
	state            State
	mu               sync.Mutex
	emu              *emulator.Emulator
	stateBroadcaster *broadcaster.Broadcaster[*protocol.AgentState]

	conn    *controller.Controller
	clients []*DisplayClient
}

func New() *Server {
	return &Server{
		state:            StateStopped,
		stateBroadcaster: broadcaster.New[*protocol.AgentState](),
	}
}

func (s *Server) Start() {
	s.broadcastState()

	go s.startHeartbeat()

	err := adb.StartServer()
	if err != nil {
		log.Fatal("failed to start adb server", err)
	}

	//conn, err := controller.Connect("127.0.0.1:8554", s.processSample)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//s.conn = conn

	lis, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	acs := &agentControllerServer{
		server: s,
	}

	grpcServer := grpc.NewServer()
	protocol.RegisterAgentControllerServer(grpcServer, acs)
	log.Fatal(grpcServer.Serve(lis))
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
	//TODO implement me
	panic("implement me")
}
