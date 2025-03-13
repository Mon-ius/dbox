#!/bin/dash

go mod init dox && go mod tidy

ARCH="amd64"
LIB_NAME="libdox.so"
OUTPUT="build/output"
TAGS="with_debug"
LDFLAGS="-s -w -buildid="

CGO_ENABLED=1 GOOS=linux GOARCH=$ARCH go build -v \
    -buildmode=c-shared \
    -trimpath \
    -buildvcs=false \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/$LIB_NAME" \
    ./src

dart run ffigen --config config.yaml 
dart compile exe src/test.dart -o "$OUTPUT/test"
chmod +x "$OUTPUT/test" && ./$OUTPUT/test

# gcc -o "$OUTPUT/test" src/test.c -L"$OUTPUT" -ldox -DDEBUG -Wl,-rpath="$OUTPUT"
# chmod +x "$OUTPUT/test" && ./$OUTPUT/test