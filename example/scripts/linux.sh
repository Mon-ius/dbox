#!/bin/dash

ARCH="amd64"
LIB_NAME="libbox.so"
OUTPUT="build/linux-$ARCH"
TAGS="with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api"
LDFLAGS="-X github.com/sagernet/sing-box/constant.Version=1.11.5 -s -w -buildid="

cd upstream && CGO_ENABLED=1 GOOS=linux GOARCH=$ARCH go build -v \
    -buildmode=c-shared \
    -trimpath \
    -buildvcs=false \
    -ldflags="$LDFLAGS" \
    -o "$OUTPUT/$LIB_NAME" \
    ./cmd/sing-box