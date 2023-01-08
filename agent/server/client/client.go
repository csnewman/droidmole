package client

import (
	"bytes"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3/pkg/media"
	"log"
)

type Client struct {
	ws           *websocket.Conn
	videoStarted bool
}

func New(ws *websocket.Conn) *Client {
	c := &Client{
		ws: ws,
	}

	ws.SetCloseHandler(c.handleClose)
	ws.WriteMessage(websocket.TextMessage, []byte("Connected"))

	return c
}

func (c *Client) handleClose(code int, text string) error {
	log.Println("Websocket closed")
	return nil
}

func (c *Client) ProcessSample(sample *media.Sample) {
	videoKeyframe := (sample.Data[0]&0x1 == 0)
	if videoKeyframe {
		c.videoStarted = true

		//raw := uint(sample.Data[6]) | uint(sample.Data[7])<<8 | uint(sample.Data[8])<<16 | uint(sample.Data[9])<<24
		//width := int(raw & 0x3FFF)
		//height := int((raw >> 16) & 0x3FFF)

		var buf bytes.Buffer
		buf.WriteByte(1)
		c.ws.WriteMessage(websocket.BinaryMessage, buf.Bytes())
	}

	if c.videoStarted {
		var buf bytes.Buffer
		buf.WriteByte(2)
		if videoKeyframe {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}

		buf.Write(sample.Data)
		c.ws.WriteMessage(websocket.BinaryMessage, buf.Bytes())
	}
}
