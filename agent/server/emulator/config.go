package emulator

import (
	"errors"
	"fmt"
	"github.com/csnewman/droidmole/agent/protocol"
	"gopkg.in/ini.v1"
	"strconv"
)

const (
	X8664    = "x86_64"
	ARM64    = "arm64"
	ARM64V8A = "arm64-v8a"
)

func GetImageCpu() (string, error) {
	cfg, err := ini.Load("/android/system-image/build.prop")
	if err != nil {
		return "", err
	}

	sect := cfg.Section("")

	if !sect.HasKey("ro.product.cpu.abi") {
		return "", errors.New("ro.product.cpu.abi missing")
	}

	key, err := sect.GetKey("ro.product.cpu.abi")
	if err != nil {
		return "", err
	}

	abi := key.Value()

	return abi, nil
}

func createConfig(request *protocol.StartEmulatorRequest) (*ini.File, error) {
	cfg := ini.Empty()
	section := cfg.Section("")

	section.Key("AvdId").SetValue("Custom")
	// CHECK: section.Key("PlayStore.enabled").SetValue("no")
	section.Key("avd.ini.displayname").SetValue("Custom")
	section.Key("avd.ini.encoding").SetValue("UTF-8")

	// Hardware
	section.Key("hw.ramSize").SetValue(strconv.FormatInt(int64(request.GetRamSize()), 10))
	section.Key("hw.cpu.ncore").SetValue(strconv.FormatInt(int64(request.GetCoreCount()), 10))
	// TODO: Replace
	section.Key("disk.dataPartition.size").SetValue("512MB")
	section.Key("fastboot.forceColdBoot").SetValue("no")
	section.Key("hw.accelerometer").SetValue("yes")
	section.Key("hw.audioInput").SetValue("yes")
	section.Key("hw.battery").SetValue("yes")
	section.Key("hw.camera.back").SetValue("emulated")
	section.Key("hw.camera.front").SetValue("emulated")
	section.Key("hw.dPad").SetValue("no")
	section.Key("hw.device.hash2").SetValue("MD5:bc5032b2a871da511332401af3ac6bb0")
	section.Key("hw.device.manufacturer").SetValue("Google")
	section.Key("hw.gps").SetValue("yes")
	section.Key("hw.gpu.enabled").SetValue("yes")
	section.Key("hw.gpu.mode").SetValue("auto")
	section.Key("hw.initialOrientation").SetValue("Portrait")
	section.Key("hw.keyboard").SetValue("yes")
	section.Key("hw.mainKeys").SetValue("no")
	section.Key("hw.sensors.orientation").SetValue("yes")
	section.Key("hw.sensors.proximity").SetValue("yes")
	section.Key("hw.trackBall").SetValue("no")
	section.Key("runtime.network.latency").SetValue("none")
	section.Key("runtime.network.speed").SetValue("full")
	// CHECK: section.Key("vm.heapSize").SetValue("512")
	// CHECK: section.Key("tag.display").SetValue("Google APIs")

	// Display
	section.Key("hw.lcd.density").SetValue(strconv.FormatInt(int64(request.GetLcdDensity()), 10))
	section.Key("hw.lcd.width").SetValue(strconv.FormatInt(int64(request.GetLcdWidth()), 10))
	section.Key("hw.lcd.height").SetValue(strconv.FormatInt(int64(request.GetLcdHeight()), 10))

	//section.Key("hw.sdCard").SetValue("yes")
	//section.Key("sdcard.size").SetValue("512M")

	abi, err := GetImageCpu()
	if err != nil {
		return nil, err
	}

	section.Key("abi.type").SetValue(abi)

	switch abi {
	case X8664:
		section.Key("hw.cpu.arch").SetValue(X8664)
	case ARM64V8A:
		section.Key("hw.cpu.arch").SetValue(ARM64)
	default:
		return nil, fmt.Errorf("unknown abi %s", abi)
	}

	section.Key("image.sysdir.1").SetValue("/android/system-image/")

	return cfg, nil
}
