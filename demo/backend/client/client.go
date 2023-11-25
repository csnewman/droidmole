package client

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"sync"

	agent "github.com/csnewman/droidmole/agent/client"
	"github.com/csnewman/droidmole/agent/client/shell"
	"github.com/gorilla/websocket"
)

type Client struct {
	ws           *websocket.Conn
	videoStarted bool
	ac           *agent.Client
	shell        *shell.Shell
	sendMutex    sync.Mutex
}

func New(ws *websocket.Conn, ac *agent.Client) *Client {
	c := &Client{
		ws: ws,
		ac: ac,
	}

	ws.SetCloseHandler(c.handleClose)

	line := "Connected"
	c.SendMsg(ClientOutMsg{
		Type: "log",
		Line: &line,
	})

	go c.processMessages()

	return c
}

type TouchEvent struct {
	Id       uint32 `json:"id"`
	X        uint32 `json:"x"`
	Y        uint32 `json:"y"`
	Pressure uint32 `json:"pressure"`
}

type ClientInMsg struct {
	Type       string      `json:"type"`
	Line       *string     `json:"line"`
	TouchEvent *TouchEvent `json:"touchEvent"`
	PowerEvent *string     `json:"powerEvent"`
}

func (c *Client) processMessages() {
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("message ", err)
			return
		}

		var msg ClientInMsg
		err = json.Unmarshal(data, &msg)
		if err != nil {
			log.Println("message ", err)
			return
		}

		switch msg.Type {
		case "open-shell":
			ctx := context.Background()
			tt := "xterm-256color"
			c.shell, err = c.ac.OpenShell(ctx, shell.Request{
				Type:     shell.TypePTY,
				Command:  nil,
				TermType: &tt,
			})
			if err != nil {
				log.Println("message ", err)
				return
			}

			go c.processShell()
		case "shell-data":
			err = c.shell.SendInput([]byte(*msg.Line))
			if err != nil {
				log.Println("message ", err)
				return
			}
		case "touch-event":
			evt := msg.TouchEvent
			ctx := context.Background()
			err = c.ac.SendInput(ctx, agent.TouchEvent{
				Identifier: evt.Id,
				X:          evt.X,
				Y:          evt.Y,
				Pressure:   evt.Pressure,
				TouchMajor: 0,
				TouchMinor: 0,
			})

			if err != nil {
				log.Println("message ", err)
				return
			}

		case "power-event":

			switch *msg.PowerEvent {
			case "start":
				ctx := context.Background()
				err := c.ac.StartEmulator(ctx, agent.StartEmulatorRequest{
					RamSize:    3500,
					CoreCount:  4,
					LcdDensity: 320,
					LcdHeight:  1280,
					LcdWidth:   720,

					//LcdHeight: 1280 / 2,
					//LcdWidth:  720 / 2,
					RootADB: true,
				})
				if err != nil {
					log.Println("message ", err)
					return
				}
			case "stop":
				ctx := context.Background()
				err := c.ac.StopEmulator(ctx, false)
				if err != nil {
					log.Println("message ", err)
					return
				}
			case "force-stop":
				ctx := context.Background()
				err := c.ac.StopEmulator(ctx, true)
				if err != nil {
					log.Println("message ", err)
					return
				}
			}
		}
	}
}

func (c *Client) ProcessFrame(frame *agent.Frame) error {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	if frame.Keyframe {
		c.videoStarted = true
		var buf bytes.Buffer
		buf.WriteByte(1)
		err := c.ws.WriteMessage(websocket.BinaryMessage, buf.Bytes())
		if err != nil {
			return err
		}
	}

	if c.videoStarted {
		var buf bytes.Buffer
		buf.WriteByte(2)
		if frame.Keyframe {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}

		buf.Write(frame.Data)
		err := c.ws.WriteMessage(websocket.BinaryMessage, buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) handleClose(code int, text string) error {
	log.Println("Websocket closed")
	return nil
}

type ClientOutMsg struct {
	Type string  `json:"type"`
	Line *string `json:"line"`
}

func (c *Client) SendMsg(msg ClientOutMsg) error {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	encoded, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return c.ws.WriteMessage(websocket.TextMessage, encoded)
}

func (c *Client) ProcessSysLog(line string) error {
	return c.SendMsg(ClientOutMsg{
		Type: "kernlog",
		Line: &line,
	})
}

func (c *Client) processShell() {
	for {
		output, err := c.shell.Recv()
		if err != nil {
			log.Println("shell error", err)
			return
		}

		if output == nil {
			log.Println("shell closed")
			return
		}

		line := string(output.Data)

		err = c.SendMsg(ClientOutMsg{
			Type: "shell-data",
			Line: &line,
		})
		if err != nil {
			log.Println("shell error", err)
			return
		}

	}
}
