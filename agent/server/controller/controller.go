package controller

import (
	"context"
	"github.com/csnewman/droidmole/agent/server/controller/protocol"
	"github.com/csnewman/droidmole/agent/server/vpx"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tmthrgd/go-shm"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"image"
	"image/color"
	"log"
	"os"
	"time"
)

type MessageCallback func(data []byte)

type Controller struct {
	connection    *grpc.ClientConn
	controlClient protocol.EmulatorControllerClient
	shmFile       *os.File
	shmData       []byte
	scrClient     protocol.EmulatorController_StreamScreenshotClient
	callback      MessageCallback
	img           *vpx.Image
	codecCtx      *vpx.CodecCtx
}

func Connect(serverUrl string, callback MessageCallback) (*Controller, error) {
	log.Println("Dialing")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(serverUrl, opts...)
	if err != nil {
		return nil, err
	}

	controlClient := protocol.NewEmulatorControllerClient(conn)

	ctx := context.Background()

	log.Println("Connecting")
	var stats *protocol.EmulatorStatus
	for {
		stats, err = controlClient.GetStatus(ctx, &empty.Empty{})
		if err != nil {
			s := status.Convert(err)
			if s.Code() == codes.Unavailable {
				time.Sleep(time.Millisecond * 50)
				continue
			}

			return nil, err
		}

		break
	}

	log.Println("Emulator", stats.Version)

	shmFile, err := shm.Open("droidmole-video", unix.O_CREAT|unix.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	// TODO: Remove hardcoding
	//section.Key("hw.lcd.height").SetValue("1280")
	//section.Key("hw.lcd.width").SetValue("720")
	width := 720
	height := 1280
	memSize := width * height * 3

	err = shmFile.Truncate(int64(memSize))
	if err != nil {
		return nil, err
	}

	shmData, err := unix.Mmap(int(shmFile.Fd()), 0, memSize, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	scrClient, err := controlClient.StreamScreenshot(ctx, &protocol.ImageFormat{
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

	log.Println("Connected")

	vp8 := vpx.VP8Iface()

	encCfg := vpx.NewCodecEncCfg()
	err = encCfg.Default(vp8)
	if err != nil {
		return nil, err
	}

	encCfg.SetGW(720)
	encCfg.SetGH(1280)
	encCfg.SetRcTargetBitrate(1_000)
	encCfg.SetGErrorResilient(1)
	encCfg.SetGTimebase(1, 60)

	codecCtx := vpx.NewCodecCtx()
	err = codecCtx.EncInit(vp8, encCfg, 0)
	if err != nil {
		return nil, err
	}

	img := vpx.NullImage().Alloc(vpx.ImageFormatI420, 720, 1280, 0)
	if img == nil {
		log.Panic("failed to create?")
	}

	controller := &Controller{
		connection:    conn,
		controlClient: controlClient,
		shmFile:       shmFile,
		shmData:       shmData,
		scrClient:     scrClient,
		callback:      callback,

		img:      img,
		codecCtx: codecCtx,
	}

	go controller.frameProcessor()

	return controller, nil
}

func (c *Controller) SendTouch(event protocol.TouchEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.controlClient.SendTouch(ctx, &event)
	return err
}

func (c *Controller) frameProcessor() {
	count := 0

	for {
		_, err := c.scrClient.Recv()
		if err != nil {
			log.Fatal("scr error", err)
		}

		data := vpx.RgbToYuv(c.shmData, 720, 1280)
		c.img.Read(data)

		flags := vpx.EFlagNone

		if count%100 == 0 {
			flags = vpx.EFlagForceKF
		}

		err = c.codecCtx.Encode(c.img, vpx.CodecPts(count), uint64(1), flags, vpx.DLRealtime)
		if err != nil {
			log.Fatal("scr error", err)
		}

		var iter vpx.CodecIter

		for {
			pkt := c.codecCtx.GetFrameBuffer(&iter)
			if pkt == nil {
				break
			}

			c.callback(pkt)
		}

		count += 1

		//buffer := new(bytes.Buffer)
		//
		//testing := &Testing{
		//	data: data,
		//}
		//
		//if err = jpeg.Encode(buffer, testing, nil); err != nil {
		//	log.Println("jpeg Encode Error", err)
		//	continue
		//}
		//
		//err = os.WriteFile("tmp/frame.jpeg", buffer.Bytes(), 0644)
		//if err != nil {
		//	panic(err)
		//}
	}

}

type Testing struct {
	data []byte
}

func (t *Testing) ColorModel() color.Model {
	//TODO implement me
	panic("implement me")
}

func (t *Testing) Bounds() image.Rectangle {
	return image.Rect(0, 0, 720, 1280)
}

func (t *Testing) At(x, y int) color.Color {
	i := ((y * 720) + x) * 3

	return color.RGBA{
		R: t.data[i],
		G: t.data[i+1],
		B: t.data[i+2],
		A: 255,
	}
}
