package server

import (
	"context"

	"github.com/csnewman/droidmole/demo/backend/client"

	agent "github.com/csnewman/droidmole/agent/client"

	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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
	syslogStream  *agent.SysLogStream
	displayStream *agent.DisplayStream
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start() {
	log.Println("Starting")

	ac, err := agent.Connect("127.0.0.1:8080")

	if err != nil {
		log.Fatal(err)
	}

	defer ac.Close()

	ctx := context.Background()

	s.displayStream, err = ac.StreamDisplay(ctx, agent.DisplayRequest{
		Format:           agent.VP8,
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
