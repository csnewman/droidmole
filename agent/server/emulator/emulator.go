package emulator

import (
	"errors"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/adb"
	"github.com/csnewman/droidmole/agent/server/emulator/controller"
	emuproto "github.com/csnewman/droidmole/agent/server/emulator/controller/protocol"
	"log"
	"os"
	"os/exec"
	"sync"
)

type Frame struct {
	Width  int
	Height int
	Data   []byte
}

type Monitor interface {
	OnEmulatorStarted()

	OnEmulatorExit(err error)

	OnEmulatorFrame(frame Frame)
}

type Emulator struct {
	monitor    Monitor
	emuCmd     *exec.Cmd
	controller *controller.Controller
	request    *protocol.StartEmulatorRequest
	mu         sync.Mutex
}

func Start(request *protocol.StartEmulatorRequest, monitor Monitor) (*Emulator, error) {
	emu := &Emulator{
		monitor: monitor,
		request: request,
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

	log.Println("Starting emulator")
	cmd := exec.Command(
		"/android/emulator/emulator",
		"-avd", "Custom",
		"-ports", "5556,5557",
		"-grpc", "8554",
		"-no-window",
		"-skip-adb-auth",
		"-no-snapshot-save",
		"-wipe-data",
		"-shell-serial", "telnet:0.0.0.0:4444,server", //,nowait
		"-logcat", "*:V",
		"-gpu", "swiftshader_indirect",
		//"-kernel", "/agent/customKern",
		"-kernel", "/android/system-image/kernel-ranchu",
		"-vendor", "/android/system-image/vendor.img",
		"-system", "/android/system-image/system.img",
		"-encryption-key", "/android/system-image/encryptionkey.img",
		//"-ramdisk", "/agent/custom.img",
		"-ramdisk", "/android/system-image/ramdisk.img",
		"-data", "/android/system-image/userdata.img",
		"-qemu", "-append", "panic=1",
	)
	cmd.Env = append(cmd.Env, "ANDROID_AVD_HOME=/android/home")
	cmd.Env = append(cmd.Env, "ANDROID_SDK_ROOT=/android")
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	err = cmd.Start()
	if err != nil {
		return err
	}

	e.emuCmd = cmd

	go e.watchEmulatorExit()

	go e.connect()

	return nil
}

func (e *Emulator) watchEmulatorExit() {
	err := e.emuCmd.Wait()
	log.Println("Emulator exited", err)
	e.monitor.OnEmulatorExit(err)
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
			Width:  int(e.request.LcdWidth),
			Height: int(e.request.LcdHeight),
			Data:   data,
		})
	}
}

func (e *Emulator) ProcessInput(event *protocol.TouchEvent) error {
	e.mu.Lock()
	controller := e.controller
	e.mu.Unlock()

	if controller == nil {
		return errors.New("emulator not ready")
	}

	touches := make([]*emuproto.Touch, 0)

	for _, e := range event.Touches {
		touches = append(touches, &emuproto.Touch{
			X:          e.X,
			Y:          e.Y,
			Identifier: e.Identifier,
			Pressure:   e.Pressure,
			TouchMajor: e.TouchMajor,
			TouchMinor: e.TouchMinor,
			Expiration: 1,
		})
	}

	return controller.SendTouch(emuproto.TouchEvent{
		Touches: touches,
		Display: 0,
	})
}
