#!/bin/dash

go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
go mod init dbox && go mod tidy

ARCH="amd64"
OUTPUT="build"
TAGS="with_debug"
LDFLAGS="-s -w -buildid="

cd src/cmd
CGO_ENABLED=1 GOOS=linux GOARCH=$ARCH go build -v \
    -buildmode=c-shared \
    -trimpath \
    -buildvcs=false \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/libdbox.so" \
    ./dbox

gomobile init 
CGO_ENABLED=1 GOOS=android GOARCH=arm64 gomobile bind -v \
    -target android \
    -androidapi 21 \
    -javapkg=io.m0nius \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/libdbox.aar" \
    .

cd ..
dart run ffigen --config ffigen.yaml
dart compile exe lib/src/dbox.dart -o "dbox-library/$OUTPUT/dbox"
chmod +x "dbox-library/$OUTPUT/dbox" && dbox-library/$OUTPUT/dbox

# gcc -o "$OUTPUT/test" src/test.c -L"$OUTPUT" -ldox -DDEBUG -Wl,-rpath="$OUTPUT"
# chmod +x "$OUTPUT/test" && ./$OUTPUT/test