FROM golang:1-alpine3.23 AS builder

RUN apk add --no-cache git ca-certificates build-base su-exec olm-dev

COPY . /build
WORKDIR /build
# Limit parallelism to reduce peak RAM usage when self-building on small VPSes.
RUN GOFLAGS="-p=1" GOMAXPROCS=1 ./build.sh

FROM alpine:3.23

ENV UID=1337 \
    GID=1337

RUN apk add --no-cache ffmpeg su-exec ca-certificates olm bash jq curl yq-go lottieconverter

COPY --from=builder /build/mautrix-telegram /usr/bin/mautrix-telegram
COPY --from=builder /build/docker-run.sh /docker-run.sh
VOLUME /data

CMD ["/docker-run.sh"]
