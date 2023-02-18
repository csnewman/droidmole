package syslog

import (
	"bufio"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
)

const SockAddr = "/tmp/sys-shell.sock"

type SysLog struct {
	listener net.Listener
	clients  []*Listener
	mu       sync.Mutex
}

func Start() (*SysLog, error) {
	err := os.RemoveAll(SockAddr)
	if err != nil {
		return nil, err
	}

	l, err := net.Listen("unix", SockAddr)
	if err != nil {
		return nil, err
	}

	s := &SysLog{
		listener: l,
		clients:  []*Listener{},
	}

	go s.processor()

	return s, nil
}

func (s *SysLog) Close() {
	s.listener.Close()
}

func (s *SysLog) processor() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Println("syslog accept error:", err)
			return
		}

		log.Println("syslog connection accepted")

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			msg := scanner.Text()

			s.mu.Lock()

			for i := len(s.clients) - 1; i >= 0; i-- {
				l := s.clients[i]

				if !l.send(msg) {
					s.clients = append(s.clients[:i], s.clients[i+1:]...)
				}
			}

			s.mu.Unlock()
		}

		if err := scanner.Err(); err != nil {
			log.Println("syslog error:", err)
		}
	}
}

type Listener struct {
	channel chan string
	closed  atomic.Bool
}

func (s *SysLog) Listen() *Listener {
	channel := make(chan string, 100)
	listener := &Listener{
		channel: channel,
	}

	s.mu.Lock()
	s.clients = append(s.clients, listener)
	s.mu.Unlock()

	return listener
}

func (l *Listener) send(msg string) bool {
	if l.closed.Load() {
		return false
	}

	select {
	case l.channel <- msg:
	default:
		log.Println("syslog listener full - dropping message")
	}

	return true
}

func (l *Listener) Recv() string {
	return <-l.channel
}

func (l *Listener) Close() {
	l.closed.Store(true)
}
