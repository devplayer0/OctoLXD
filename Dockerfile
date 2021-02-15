FROM golang:1.15-alpine AS builder
RUN apk --no-cache add gcc musl-dev linux-headers

WORKDIR /usr/local/src/octolxd
COPY go.* ./
RUN go mod download

COPY tools.go ./
RUN cat tools.go | sed -nr 's|^\t_ "(.+)"$|\1|p' | xargs -tI % go get %

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
RUN mkdir bin/ && go build -o bin/ ./cmd/...


FROM alpine:3.13

COPY --from=builder /usr/local/src/octolxd/bin/* /usr/local/bin/

EXPOSE 80/tcp
ENTRYPOINT ["/usr/local/bin/octolxd"]

LABEL org.opencontainers.image.source https://github.com/devplayer0/octolxd
