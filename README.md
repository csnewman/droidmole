# droidmole

Android interception framework

## Build (Container) Agent

Install dependencies:

1. Install Go
2. Install necessary packages
   ```
   apt-get install libarchive-tools
   ```

Build `dmb` (DroidMole Builder) utility:

```
cd builder
go build -o ../dmb
cd ..
```

Download Android components:

1. Android Emulator
   ```
   ./dmb download emulator --output agent/prebuilts/emulator.zip
   ```
2. Android Platform Tools
   ```
   ./dmb download platform-tools --output agent/prebuilts/tools.zip
   ```
3. Android Image
   ```
   ./dmb dmb download sysimg --api 33 --type google --output agent/prebuilts/sysimg.zip
   ```

Build container:

```
cd agent
DOCKER_BUILDKIT=1 docker build -t droidmole-android33 .
```

## Run Agent

```
docker run --rm --name android1 --device /dev/kvm droidmole-android33
```

The agent will now be hosting a gRPC server on port 8080.

## Development

1. Install Go
2. Install necessary packages
   ```
   apt-get install libarchive-tools libvpx-dev build-essential pkg-config
   ```
