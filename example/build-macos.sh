#!/bin/dash

ARCH="amd64"
OUTPUT="build"
TAGS="with_debug"
LDFLAGS="-s -w -buildid="

cd external
CGO_ENABLED=1 GOOS=darwin GOARCH=$ARCH go build -v \
    -buildmode=c-shared \
    -trimpath \
    -buildvcs=false \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/libdbox.dylib" \
    ./cmd/dbox

cd ..
dart run ffigen --config ffigen.yaml
dart compile exe lib/src/dbox.dart -o "external/$OUTPUT/dbox"
chmod +x "external/$OUTPUT/dbox" && external/$OUTPUT/dbox

# gcc -o "$OUTPUT/test" external/test.c -L"$OUTPUT" -ldox -DDEBUG -Wl,-rpath="$OUTPUT"
# chmod +x "$OUTPUT/test" && ./$OUTPUT/test

gomobile bind -v \
    -target ios \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/Dbox.framework" \
    ./mobile