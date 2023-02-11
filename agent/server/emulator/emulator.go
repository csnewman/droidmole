package emulator

import (
	"github.com/csnewman/droidmole/agent/protocol"
	"log"
	"os"
	"os/exec"
)

type Monitor interface {
	OnEmulatorExit(err error)
}

type Emulator struct {
	monitor Monitor
}

func Start(request *protocol.StartEmulatorRequest, monitor Monitor) *Emulator {
	emu := &Emulator{
		monitor: monitor,
	}
	go emu.start(request)
	return emu
}

func (e *Emulator) start(request *protocol.StartEmulatorRequest) {
	err := e.startInner(request)
	e.monitor.OnEmulatorExit(err)
}

func (e *Emulator) startInner(request *protocol.StartEmulatorRequest) error {
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
	cfg, err := createConfig(request)
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
	err = cmd.Run()
	if err != nil {
		return err
	}

	log.Println("Emulator exited")
	return nil
}
