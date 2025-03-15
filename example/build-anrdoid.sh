#!/bin/dash

GOOS=android
OUTPUT="../platforms/android/src/main/jniLibs"
TAGS="with_debug"
LDFLAGS="-s -w -buildid="
ANDROID_SDK=$HOME/Android/Sdk
NDK_BIN=$ANDROID_SDK/ndk/toolchains/llvm/prebuilt/linux-x86_64/bin

echo "Building for arm64-v8a..."
ARCH="arm64"
CGO_ENABLED=1 GOOS=$GOOS GOARCH=$ARCH \
    CC=$NDK_BIN/aarch64-linux-android21-clang \
    go build -buildmode=c-shared \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/arm64-v8a/libdbox.so" \
    ./cmd/dbox

echo "Building for armeabi-v7a..."
ARCH="arm"
CGO_ENABLED=1 GOOS=$GOOS GOARCH=$ARCH GOARM=7 \
    CC=$NDK_BIN/armv7a-linux-androideabi21-clang \
    go build -buildmode=c-shared \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/armeabi-v7a/libdbox.so" \
    ./cmd/dbox

echo "Building for x86_64..."
ARCH="amd64"
CGO_ENABLED=1 GOOS=android GOARCH=amd64 \
    CC=$NDK_BIN/x86_64-linux-android21-clang \
    go build -buildmode=c-shared \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/x86_64/libdbox.so" \
    ./cmd/dbox

echo "Building for x86..."
ARCH="386"
CGO_ENABLED=1 GOOS=android GOARCH=$ARCH \
    CC=$NDK_BIN/i686-linux-android21-clang \
    go build -buildmode=c-shared \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -tags="$TAGS" \
    -o "$OUTPUT/x86/libdbox.so" \
    ./cmd/dbox