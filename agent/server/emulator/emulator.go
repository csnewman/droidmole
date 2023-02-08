package emulator

import (
	"gopkg.in/ini.v1"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func CreateConfig() *ini.File {
	cfg := ini.Empty()
	section := cfg.Section("")

	section.Key("AvdId").SetValue("Custom")
	section.Key("PlayStore.enabled").SetValue("no")
	section.Key("avd.ini.displayname").SetValue("Pixel2")
	section.Key("avd.ini.encoding").SetValue("UTF-8")

	// TODO: Replace
	section.Key("hw.ramSize").SetValue("2048")
	section.Key("hw.cpu.ncore").SetValue("1")

	// Real Pixel2 ships with 32GB
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
	section.Key("vm.heapSize").SetValue("512")
	section.Key("tag.display").SetValue("Google APIs")

	// TODO: Replace
	//section.Key("hw.lcd.density").SetValue("440")
	//section.Key("hw.lcd.height").SetValue("1920")
	//section.Key("hw.lcd.width").SetValue("1080")
	section.Key("hw.lcd.density").SetValue("320")
	section.Key("hw.lcd.height").SetValue("1280")
	section.Key("hw.lcd.width").SetValue("720")

	// Unused
	//section.Key("hw.sdCard").SetValue("yes")
	//section.Key("sdcard.size").SetValue("512M")

	// TODO: Replace
	section.Key("abi.type").SetValue("x86_64")
	section.Key("hw.cpu.arch").SetValue("x86_64")
	section.Key("image.sysdir.1").SetValue("/android/system-image/")

	return cfg
}

func Run() {
	log.Println("Hello world")

	err := os.MkdirAll("/android/home/Custom.avd", 0644)
	if err != nil {
		log.Fatal(err)
	}

	d1 := []byte("path=/android/home/Custom.avd")
	err = os.WriteFile("/android/home/Custom.ini", d1, 0644)
	if err != nil {
		log.Fatal(err)
	}

	cfg := CreateConfig()
	cfg.SaveTo("/android/home/Custom.avd/config.ini")

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(dirname)

	err = os.MkdirAll(filepath.Join(dirname, ".android"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	keyPath := filepath.Join(dirname, ".android/adbkey")

	log.Println("Generating", keyPath)

	cmd := exec.Command("/android/platform-tools/adb", "keygen", keyPath)
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chmod(keyPath, 0600)
	if err != nil {
		log.Fatal(err)
	}

	cmd = exec.Command(
		"/android/emulator/emulator",
		"-avd", "Custom",
		"-ports", "5556,5557",
		"-grpc", "8554",
		"-no-window",
		"-skip-adb-auth",
		"-no-snapshot-save",
		"-wipe-data",
		//"-no-boot-anim",
		"-logcat", "*:V",
		"-gpu", "swiftshader_indirect",
		"-qemu", "-append", "panic=1",
	)
	cmd.Env = append(cmd.Env, "ANDROID_AVD_HOME=/android/home")
	cmd.Env = append(cmd.Env, "ANDROID_SDK_ROOT=/android")

	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Emulator exited")
}
