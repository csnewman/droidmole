package server

import (
	"droidmole/server/client"
	"droidmole/server/connection"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3/pkg/media"
	"log"
	"net/http"
	"os"
	"sync"
)

var upgrader = websocket.Upgrader{}

type Server struct {
	conn    *connection.CfConnection
	mu      sync.Mutex
	clients []*client.Client
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start() {
	conn, err := connection.New(os.Args[1], s.processSample)
	if err != nil {
		log.Fatal(err)
	}

	s.conn = conn

	http.HandleFunc("/ws", s.wsEndpoint)
	http.Handle("/", http.FileServer(http.Dir("webroot/")))
	//log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
	log.Fatal(http.ListenAndServeTLS("0.0.0.0:8080", "localhost.crt", "localhost.key", nil))
}

func (s *Server) wsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	c := client.New(ws)

	s.mu.Lock()
	s.clients = append(s.clients, c)
	s.mu.Unlock()
}

func (s *Server) processSample(sample *media.Sample) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.clients {
		c.ProcessSample(sample)
	}
}
