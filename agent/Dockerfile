FROM ghcr.io/csnewman/droidmole/avd-resources:dev-latest as avd-resources
FROM ubuntu:23.04 AS builder

ARG TARGETARCH

# Install build dependencies
RUN apt-get update && \
    apt-get install -y curl libvpx-dev ca-certificates libarchive-tools build-essential pkg-config adb

# Download golang
RUN curl -s https://dl.google.com/go/go1.20.linux-${TARGETARCH}.tar.gz | tar -v -C /usr/local -xz
ENV PATH $PATH:/usr/local/go/bin

# Extract emulator
RUN mkdir /emulator/
COPY --from=avd-resources /emulator-${TARGETARCH}.zip /tmp/emulator.zip
RUN bsdtar -xf /tmp/emulator.zip --strip-components=1 -C /emulator

# ARM builds appear to be missing vulkan libs
# Pull from
# [ "$TARGETARCH" = "arm64" ] &&
RUN if [ ! -d /emulator/lib64/vulkan/ ] ; then \
      mkdir /armhack/ && \
      mkdir -p /emulator/lib64/vulkan && \
      cd /armhack/ && \
      apt-get install -y glslang-tools snapd && \
      cp $(which glslangValidator) /emulator/lib64/vulkan && \
      snap download --basename=chromium chromium && \
      unsquashfs -d chromium chromium.snap && \
      cp ./chromium/usr/lib/chromium-browser/libvulkan.so.1 /emulator/lib64/vulkan/libvulkan.so && \
      cp ./chromium/usr/lib/chromium-browser/libvk_swiftshader.so /emulator/lib64/vulkan/libvk_swiftshader.so && \
      cp ./chromium/usr/lib/chromium-browser/vk_swiftshader_icd.json /emulator/lib64/vulkan \
    ; fi

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

FROM ubuntu:23.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        # emulator dependencies
        libc6 libdbus-1-3 libfontconfig1 libgcc1 libtinfo5 libx11-6 libxcb1 libxdamage1 libnss3 libxcomposite1 \
        libxcursor1 libxi6 libxext6 libxfixes3 zlib1g libgl1 adb \
        # agent dependencies
        libvpx7 && \
    apt-get clean  && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN mkdir -p /android/platforms && \
    mkdir -p /android/platform-tools && \
    ln -s $(which adb) /android/platform-tools/adb

# Copy build output
COPY --from=builder /emulator /android/emulator
COPY --from=builder /out/agent /agent/agent

WORKDIR /agent
CMD ["/agent/agent"]
