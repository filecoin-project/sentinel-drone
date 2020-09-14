# Tweaked from ./scripts/stretch.docker
FROM golang:1.13.8 as builder

RUN apt-get update
# Install deps for filecoin-project/filecoin-ffi
RUN apt-get install -y jq mesa-opencl-icd ocl-icd-opencl-dev

WORKDIR /go/src/github.com/influxdata/telegraf
COPY . /go/src/github.com/influxdata/telegraf
RUN make go-install

# Updated from stretch to buster, fixes incorrect glib version error from filecoin-ffi build
FROM buildpack-deps:buster-curl
# Grab the things
COPY --from=builder /go/bin/* /usr/bin/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libOpenCL.so* /lib/

COPY etc/telegraf.conf /etc/telegraf/telegraf.conf

EXPOSE 8125/udp 8092/udp 8094

COPY scripts/docker-entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
CMD ["telegraf"]
