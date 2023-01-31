package server

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/controller"
	emuproto "github.com/csnewman/droidmole/agent/server/controller/protocol"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
	"time"
)

type Server struct {
	conn    *controller.Controller
	mu      sync.Mutex
	clients []*DisplayClient
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start() {
	conn, err := controller.Connect("127.0.0.1:8554", s.processSample)
	if err != nil {
		log.Fatal(err)
	}

	s.conn = conn

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

type agentControllerServer struct {
	protocol.UnimplementedAgentControllerServer
	server *Server
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
