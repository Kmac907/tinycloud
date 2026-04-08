FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -o /out/tinycloud ./cmd/tinycloud && \
    go build -o /out/tinycloudd ./cmd/tinycloudd

FROM alpine:3.21
RUN adduser -D -u 10001 tinycloud && mkdir -p /var/lib/tinycloud && chown -R tinycloud:tinycloud /var/lib/tinycloud
USER tinycloud
WORKDIR /app

COPY --from=build /out/tinycloud /usr/local/bin/tinycloud
COPY --from=build /out/tinycloudd /usr/local/bin/tinycloudd

ENV TINYCLOUD_DATA_ROOT=/var/lib/tinycloud

EXPOSE 4566 4567 4577 4578 4579 4580 4581 4582 4583 4584/udp 4585

CMD ["tinycloudd"]
