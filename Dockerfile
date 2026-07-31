FROM golang:1.25-alpine AS build

ARG GOPROXY=https://proxy.golang.org,direct
WORKDIR /src
COPY go.mod go.sum ./
RUN GOPROXY="${GOPROXY}" go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sshw-server ./cmd/sshw-server

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 sshw \
    && adduser -S -D -H -u 10001 -G sshw sshw \
    && mkdir -p /data \
    && chown -R sshw:sshw /data

COPY --from=build /out/sshw-server /usr/local/bin/sshw-server
COPY --chmod=755 docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

USER sshw
VOLUME ["/data"]
EXPOSE 8080

ENV SSHW_SERVER_ADDR=:8080
ENV SSHW_DATABASE_PATH=/data/sshw.db

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/usr/local/bin/sshw-server"]
