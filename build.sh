#!/bin/dash

go mod init dox && go mod tidy

ARCH="amd64"
LIB_NAME="libdox.so"
OUTPUT="build"
TAGS="with_debug"
LDFLAGS="-s -w -buildid="

cd dox-library
CGO_ENABLED=1 GOOS=linux GOARCH=$ARCH go build -v \
    -buildmode=c-shared \
    -trimpath \
    -buildvcs=false \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/$LIB_NAME" \
    .

cd ..
dart run ffigen --config ffigen.yaml
dart compile exe lib/src/dox.dart -o "dox-library/$OUTPUT/dox"
chmod +x "dox-library/$OUTPUT/dox" && dox-library/$OUTPUT/dox

# gcc -o "$OUTPUT/test" src/test.c -L"$OUTPUT" -ldox -DDEBUG -Wl,-rpath="$OUTPUT"
# chmod +x "$OUTPUT/test" && ./$OUTPUT/test