FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

RUN addgroup -g 65532 -S nonroot && \
    adduser  -u 65532 -S -D -G nonroot -s /sbin/nologin nonroot

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG GOOS=linux
ARG GOARCH=amd64

ENV GOOS=$GOOS
ENV GOARCH=$GOARCH

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/server ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/server /server

ENV VINYL_BIND_ADDR=0.0.0.0:8080

EXPOSE 8080

USER nonroot

ENTRYPOINT ["/server"]
