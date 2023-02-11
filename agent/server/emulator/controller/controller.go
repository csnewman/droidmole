package controller

import (
	"context"
	"github.com/csnewman/droidmole/agent/server/emulator/controller/protocol"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tmthrgd/go-shm"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
	"time"
)

type Controller struct {
	connection    *grpc.ClientConn
	controlClient protocol.EmulatorControllerClient
}

func Connect(serverUrl string) (*Controller, error) {
	log.Println("Connecting to grpc")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(serverUrl, opts...)
	if err != nil {
		return nil, err
	}

	controlClient := protocol.NewEmulatorControllerClient(conn)

	ctx := context.Background()
	stats, err := controlClient.GetStatus(ctx, &empty.Empty{})
	if err != nil {
		return nil, err
	}

	log.Println("Connected to grpc")
	log.Println("Emulator version:", stats.Version)

	controller := &Controller{
		connection:    conn,
		controlClient: controlClient,
	}

	return controller, nil
}

func (c *Controller) SendTouch(event protocol.TouchEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.controlClient.SendTouch(ctx, &event)
	return err
}

type DisplayStream struct {
	shmFile   *os.File
	shmData   []byte
	scrClient protocol.EmulatorController_StreamScreenshotClient
}

func (c *Controller) StreamDisplay(width int, height int) (*DisplayStream, error) {
	shmFile, err := shm.Open("droidmole-video", unix.O_CREAT|unix.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	memSize := width * height * 3
	err = shmFile.Truncate(int64(memSize))
	if err != nil {
		return nil, err
	}

	shmData, err := unix.Mmap(int(shmFile.Fd()), 0, memSize, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	scrClient, err := c.controlClient.StreamScreenshot(ctx, &protocol.ImageFormat{
		Format:   protocol.ImageFormat_RGB888,
		Rotation: nil,
		Width:    uint32(width),
		Height:   uint32(height),
		Display:  0,
		Transport: &protocol.ImageTransport{
			Channel: protocol.ImageTransport_MMAP,
			Handle:  "droidmole-video",
		},
		FoldedDisplay: nil,
		DisplayMode:   0,
	})
	if err != nil {
		return nil, err
	}

	return &DisplayStream{
		shmFile:   shmFile,
		shmData:   shmData,
		scrClient: scrClient,
	}, nil
}

func (ds *DisplayStream) GetFrame() ([]byte, error) {
	_, err := ds.scrClient.Recv()
	if err != nil {
		return nil, err
	}

	frame := make([]byte, len(ds.shmData))
	copy(frame, ds.shmData)

	return frame, nil
}
