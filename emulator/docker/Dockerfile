FROM ubuntu:jammy AS builder
USER root
WORKDIR /root

RUN apt-get update
RUN apt-get --no-install-recommends -y install git golang curl ca-certificates devscripts config-package-dev debhelper-compat

COPY build-pkgs.sh ./
RUN ./build-pkgs.sh

FROM ubuntu:jammy
ENV container docker
ENV LC_ALL C
ENV DEBIAN_FRONTEND noninteractive
USER root
WORKDIR /root

RUN apt-get update
RUN apt-get --no-install-recommends -y install iproute2

RUN if test $(uname -m) == aarch64; then \
	    dpkg --add-architecture amd64 \
	    && apt-get update \
	    && apt-get install --no-install-recommends -y libc6:amd64 \
	    && apt-get install --no-install-recommends -y qemu qemu-user qemu-user-static binfmt-support; \
    fi

COPY --from=builder /packages/*.deb /packages/
RUN cd /packages \
    && apt-get install --no-install-recommends -y -f ./cuttlefish-base_*.deb \
    && apt-get install --no-install-recommends -y -f ./cuttlefish-user_*.deb \
    && apt-get install --no-install-recommends -y -f ./cuttlefish-common_*.deb \
    && cd /root \
    && rm -rvf /packages

RUN apt-get install -y bash
RUN apt-get clean

RUN usermod -aG cvdnetwork root
RUN usermod -aG kvm root

WORKDIR /device
CMD ["bash"]
