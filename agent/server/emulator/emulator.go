package emulator

import (
	"bufio"
	"fmt"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/adb"
	"github.com/csnewman/droidmole/agent/server/emulator/controller"
	emuproto "github.com/csnewman/droidmole/agent/server/emulator/controller/protocol"
	"github.com/csnewman/droidmole/agent/server/syslog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Frame struct {
	Width  uint32
	Height uint32
	Data   []byte
}

type Monitor interface {
	OnEmulatorStarted()

	OnEmulatorExit(err error)

	OnEmulatorFrame(frame Frame)
}

type Emulator struct {
	monitor       Monitor
	emuCmd        *exec.Cmd
	controller    *controller.Controller
	request       *protocol.StartEmulatorRequest
	mu            sync.Mutex
	outPipeReader *io.PipeReader
	outPipeWriter *io.PipeWriter
	errPipeReader *io.PipeReader
	errPipeWriter *io.PipeWriter
	exitErr       chan string
}

func Start(request *protocol.StartEmulatorRequest, monitor Monitor) (*Emulator, error) {
	opr, opw := io.Pipe()
	epr, epw := io.Pipe()

	emu := &Emulator{
		monitor:       monitor,
		request:       request,
		outPipeReader: opr,
		outPipeWriter: opw,
		errPipeReader: epr,
		errPipeWriter: epw,
		exitErr:       make(chan string),
	}
	err := emu.startEmulator()
	if err != nil {
		return nil, err
	}

	return emu, nil
}

func (e *Emulator) startEmulator() error {
	log.Println("Starting emulator")

	// Create emulator directories
	err := os.MkdirAll("/android/home/Custom.avd", 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile("/android/home/Custom.ini", []byte("path=/android/home/Custom.avd"), 0644)
	if err != nil {
		return err
	}

	// Create emulator config
	cfg, err := createConfig(e.request)
	if err != nil {
		return err
	}

	err = cfg.SaveTo("/android/home/Custom.avd/config.ini")
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"/android/emulator/emulator",
		"-avd", "Custom",
		"-ports", "5556,5557",
		"-grpc", "8554",
		"-no-window",
		"-skip-adb-auth",
		"-no-snapshot-save",
		"-wipe-data",
		"-shell-serial", fmt.Sprintf("unix:%s", syslog.SockAddr),
		"-gpu", "swiftshader_indirect",
		//"-debug", "all",
		// TODO: Add image overriding
		//"-kernel", "/android/system-image/kernel-ranchu",
		//"-vendor", "/android/system-image/vendor.img",
		//"-system", "/android/system-image/system.img",
		//"-encryption-key", "/android/system-image/encryptionkey.img",
		//"-ramdisk", "/android/system-image/ramdisk.img",
		//"-data", "/android/system-image/userdata.img",
		"-qemu", "-append", "panic=1",
	)
	cmd.Env = append(cmd.Env, "ANDROID_AVD_HOME=/android/home")
	cmd.Env = append(cmd.Env, "ANDROID_SDK_ROOT=/android")
	cmd.Stdout = e.outPipeWriter
	cmd.Stderr = e.errPipeWriter

	err = cmd.Start()
	if err != nil {
		return err
	}

	e.emuCmd = cmd

	go e.processLogs()

	go e.watchEmulatorExit()

	go e.connect()

	return nil
}

func (e *Emulator) processLogs() {
	outChan := make(chan string)
	errChan := make(chan string)

	go processChannel(e.outPipeReader, outChan)
	go processChannel(e.errPipeReader, errChan)

	lastError := ""

outer:
	for {
		select {
		case line, ok := <-outChan:
			if !ok {
				break outer
			}
			log.Println("[OUT]", line)
		case line, ok := <-errChan:
			if !ok {
				break outer
			}
			log.Println("[ERR]", line)

			if strings.HasPrefix(line, "ERROR   |") {
				lastError = line
			} else {
				lastError += "\n" + line
			}
		}
	}

	log.Println("Waiting for end of stdout")
	for {
		line, ok := <-outChan
		if !ok {
			break
		}
		log.Println("[OUT]", line)
	}

	log.Println("Waiting for end of stderr")
	for {
		line, ok := <-errChan
		if !ok {
			break
		}
		log.Println("[ERR]", line)

		if strings.HasPrefix(line, "ERROR   |") {
			lastError = strings.TrimPrefix(line, "ERROR   |")
		} else {
			lastError += "\n" + line
		}
	}

	log.Println("Emulator output end reached")
	e.exitErr <- strings.TrimSpace(lastError)
}

func processChannel(reader *io.PipeReader, dstChan chan string) {
	defer func() {
		// recover from panic caused by writing to a closed channel
		if r := recover(); r != nil {
			log.Println("logs error recovered", r)
			return
		}
	}()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		dstChan <- line
	}

	reader.Close()
	close(dstChan)
}

func (e *Emulator) watchEmulatorExit() {
	err := e.emuCmd.Wait()
	log.Println("Emulator exited", err)

	e.outPipeWriter.Close()
	e.errPipeWriter.Close()

	lastError := <-e.exitErr

	var finalError error
	if err != nil {
		finalError = fmt.Errorf("emulator exited with: %s, last error: %s", err, lastError)
	}

	e.monitor.OnEmulatorExit(finalError)
}

func (e *Emulator) connect() {
	conn, err := controller.Connect(":8554")
	if err != nil {
		log.Fatal(err)
	}

	e.mu.Lock()
	e.controller = conn
	e.mu.Unlock()

	go e.processDisplay()

	log.Println("Waiting for ADB connection")
	err = adb.WaitForEmulator()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Emulator started")
	e.monitor.OnEmulatorStarted()
}

func (e *Emulator) processDisplay() {
	display, err := e.controller.StreamDisplay(int(e.request.LcdWidth), int(e.request.LcdHeight))
	if err != nil {
		log.Println("Display connection lost")
		return
	}

	for {
		data, err := display.GetFrame()
		if err != nil {
			log.Println("Display connection lost")
			return
		}

		e.monitor.OnEmulatorFrame(Frame{
			Width:  e.request.LcdWidth,
			Height: e.request.LcdHeight,
			Data:   data,
		})
	}
}

func (e *Emulator) ProcessInput(request protocol.InputRequest) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.controller == nil {
		return status.Errorf(codes.FailedPrecondition, "emulator not ready")
	}

	switch event := request.Event.(type) {
	case *protocol.InputRequest_Touch:
		return e.controller.SendTouch(emuproto.TouchEvent{
			Touches: []*emuproto.Touch{
				{
					X:          int32(event.Touch.X),
					Y:          int32(event.Touch.Y),
					Identifier: int32(event.Touch.Identifier),
					Pressure:   int32(event.Touch.Pressure),
					TouchMajor: event.Touch.TouchMajor,
					TouchMinor: event.Touch.TouchMinor,
					Expiration: emuproto.Touch_NEVER_EXPIRE,
				},
			},
			Display: 0,
		})
	default:
		return status.Errorf(codes.InvalidArgument, "unknown request")
	}
}
