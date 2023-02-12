FROM ubuntu:22.04 AS builder

# Install build dependencies
RUN apt-get update && \
    apt-get install -y curl libvpx-dev ca-certificates libarchive-tools build-essential pkg-config

# Download golang
RUN curl -s https://dl.google.com/go/go1.20.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV PATH $PATH:/usr/local/go/bin

# Extract prebuilts
COPY prebuilts/sysimg.zip /tmp/
COPY prebuilts/emulator.zip /tmp/
COPY prebuilts/tools.zip /tmp/

RUN mkdir /sysimg/
RUN mkdir /emulator/
RUN mkdir /tools/

RUN bsdtar -xf /tmp/sysimg.zip --strip-components=1 -C /sysimg
RUN bsdtar -xf /tmp/emulator.zip --strip-components=1 -C /emulator
RUN bsdtar -xf /tmp/tools.zip --strip-components=1 -C /tools

# Build agent
WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Build
COPY bin bin
COPY protocol protocol
COPY server server
COPY util util
RUN go build -v -o /out/agent ./bin/server/main.go

FROM ubuntu:22.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        # emulator dependencies
        libc6 libdbus-1-3 libfontconfig1 libgcc1 \
        libtinfo5 libx11-6 libxcb1 libxdamage1 \
        libnss3 libxcomposite1 libxcursor1 libxi6 \
        libxext6 libxfixes3 zlib1g libgl1 \
        # agent dependencies
        libvpx7 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN mkdir -p /android/platforms

# Copy build output
COPY --from=builder /sysimg/ /android/system-image
COPY --from=builder /tools/adb /android/platform-tools/adb
COPY --from=builder /emulator /android/emulator
COPY --from=builder /out/agent /agent/agent
RUN chmod +x /android/platform-tools/adb

WORKDIR /agent
CMD ["/agent/agent"]