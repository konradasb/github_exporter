FROM golang:1.17 as build

ARG GITHUB_PRIVATE_KEY="" \
    GITHUB_APP_ID="" \
    GITHUB_INS_ID=""

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOARCH=amd64 \
    GOOS=linux

RUN \
  addgroup --system github_exporter && \
  adduser --system --uid 1001 --group github_exporter

WORKDIR /build

COPY go.mod ./
COPY go.sum ./

RUN go mod download
RUN go mod verify

COPY . .

RUN go build -o github_exporter .

RUN apt update && apt install xz-utils -y && \
    strip /build/github_exporter && \
    wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz && \
    tar xf upx-3.96-amd64_linux.tar.xz && \
    ./upx-3.96-amd64_linux/upx /build/github_exporter

FROM alpine:3

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group

USER github_exporter:github_exporter

COPY --from=build --chown=github_exporter:github_exporter /build/github_exporter /

EXPOSE 9024

ENTRYPOINT [ "/github_exporter" ]
