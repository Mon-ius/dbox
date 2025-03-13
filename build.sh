#!/bin/dash

go mod init dbox && go mod tidy

ARCH="amd64"
LIB_NAME="libdbox.so"
OUTPUT="build"
TAGS="with_debug"
LDFLAGS="-s -w -buildid="

cd dbox-library
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
dart compile exe lib/src/dbox.dart -o "dbox-library/$OUTPUT/dbox"
chmod +x "dbox-library/$OUTPUT/dbox" && dbox-library/$OUTPUT/dbox

# gcc -o "$OUTPUT/test" src/test.c -L"$OUTPUT" -ldox -DDEBUG -Wl,-rpath="$OUTPUT"
# chmod +x "$OUTPUT/test" && ./$OUTPUT/test