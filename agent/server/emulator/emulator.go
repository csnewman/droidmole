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
	"syscall"
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
	monitor            Monitor
	emuCmd             *exec.Cmd
	controller         *controller.Controller
	request            *protocol.StartEmulatorRequest
	mu                 sync.Mutex
	outPipeReader      *io.PipeReader
	outPipeWriter      *io.PipeWriter
	errPipeReader      *io.PipeReader
	errPipeWriter      *io.PipeWriter
	exitErr            chan string
	forceExitRequested bool
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

	// Redirect output
	cmd.Stdout = e.outPipeWriter
	cmd.Stderr = e.errPipeWriter

	// Place emulator into its own process group to allow terminating of entire process tree
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
	inError := false

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
				inError = true
				lastError = strings.TrimPrefix(line, "ERROR   |")
			} else if strings.HasPrefix(line, "WARNING |") || strings.HasPrefix(line, "INFO    |") {
				inError = false
			} else if inError {
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
			inError = true
			lastError = strings.TrimPrefix(line, "ERROR   |")
		} else if strings.HasPrefix(line, "WARNING |") || strings.HasPrefix(line, "INFO    |") {
			inError = false
		} else if inError {
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

	status := e.emuCmd.ProcessState.Sys().(syscall.WaitStatus)

	lastError := <-e.exitErr

	var finalError error
	if e.forceExitRequested && status.Signaled() && status.Signal() == syscall.SIGKILL {
		// Don't treat a requested force kill as an error
	} else if err != nil {
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

	// Root the connection if requested
	if e.request.RootAdb {
		emuCon, err := adb.OpenEmulator()
		if err != nil {
			log.Fatal(err)
		}

		defer emuCon.Close()

		err = emuCon.SendCommand([]byte("root:"))
		if err != nil {
			log.Fatal(err)
		}

		line, err := emuCon.ReadLine()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Root response:", *line)
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

func (e *Emulator) Stop(request *protocol.StopEmulatorRequest) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Kill by terminating process group
	if request.ForceExit {
		e.forceExitRequested = true
		return syscall.Kill(-e.emuCmd.Process.Pid, syscall.SIGKILL)
	}

	if e.controller == nil {
		return status.Errorf(codes.FailedPrecondition, "emulator not ready")
	}

	return e.controller.RequestExit()
}
