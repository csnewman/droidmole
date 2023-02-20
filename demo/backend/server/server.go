package server

import (
	"context"
	"github.com/csnewman/droidmole/agent/client/display"
	"github.com/csnewman/droidmole/agent/client/state"
	"github.com/csnewman/droidmole/agent/client/syslog"
	"github.com/csnewman/droidmole/demo/backend/client"

	agent "github.com/csnewman/droidmole/agent/client"

	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	mu            sync.Mutex
	clients       []*client.Client
	ac            *agent.Client
	syslogStream  *syslog.Stream
	displayStream *display.Stream
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start() {
	log.Println("Starting")

	ac, err := agent.Connect("172.17.0.2:8080")
	if err != nil {
		log.Fatal(err)
	}

	defer ac.Close()

	ctx := context.Background()

	ss, err := ac.StreamState(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	s.displayStream, err = ac.StreamDisplay(ctx, display.Request{
		Format:           display.VP8,
		MaxFPS:           20,
		KeyframeInterval: 3000,
	})
	if err != nil {
		return
	}

	go s.processDisplay()

	s.syslogStream, err = ac.StreamSysLog(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	go s.processSysLog()

	initialState, err := ss.Recv()
	if err != nil {
		log.Fatal(err)
	}

	if initialState.EmulatorState == state.EmulatorOff {
		log.Println("Booting emulator")
		err = ac.StartEmulator(ctx, agent.StartEmulatorRequest{
			RamSize:    4096,
			CoreCount:  6,
			LcdDensity: 320,
			LcdHeight:  1280,
			LcdWidth:   720,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	s.ac = ac

	log.Println("Starting web server")

	http.HandleFunc("/ws123", s.wsEndpoint)
	http.Handle("/", http.FileServer(http.Dir("webroot/")))
	//log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
	log.Fatal(http.ListenAndServeTLS("0.0.0.0:8080", "localhost.crt", "localhost.key", nil))
}

func (s *Server) wsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	c := client.New(ws, s.ac)

	s.mu.Lock()
	s.clients = append(s.clients, c)
	s.mu.Unlock()
}

func (s *Server) processDisplay() {
	for {
		frame, err := s.displayStream.Recv()
		if err != nil {
			log.Fatal(err)
		}

		s.mu.Lock()

		for _, c := range s.clients {
			err = c.ProcessFrame(frame)
			if err != nil {
				log.Println(err)
			}
		}

		s.mu.Unlock()
	}
}

func (s *Server) processSysLog() {

	for {
		msg, err := s.syslogStream.Recv()
		if err != nil {
			log.Fatal(err)
		}

		s.mu.Lock()

		for _, c := range s.clients {
			err := c.ProcessSysLog(msg.Line)
			if err != nil {
				log.Println(err)
			}
		}

		s.mu.Unlock()
	}
}
