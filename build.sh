#!/bin/dash

go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
go mod init dbox && go mod tidy

ARCH="amd64"
OUTPUT="build"
TAGS="with_debug"
LDFLAGS="-s -w -buildid="

cd src
CGO_ENABLED=1 GOOS=linux GOARCH=$ARCH go build -v \
    -buildmode=c-shared \
    -trimpath \
    -buildvcs=false \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/libdbox.so" \
    ./cmd/dbox

go install golang.org/x/mobile/cmd/gomobile@latest
go get golang.org/x/mobile/bind
gomobile init 
gomobile bind -v \
    -target android \
    -androidapi 21 \
    -javapkg=io.m0nius \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/libdbox.aar" \
    ./mobile

cd ..
dart run ffigen --config ffigen.yaml
dart compile exe lib/src/dbox.dart -o "src/$OUTPUT/dbox"
chmod +x "src/$OUTPUT/dbox" && src/$OUTPUT/dbox

# gcc -o "$OUTPUT/test" src/test.c -L"$OUTPUT" -ldox -DDEBUG -Wl,-rpath="$OUTPUT"
# chmod +x "$OUTPUT/test" && ./$OUTPUT/test

gomobile bind -v \
    -target ios \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/Dbox.framework" \
    ./mobile