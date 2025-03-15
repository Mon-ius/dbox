//go:build with_debug

package main

// #include <stdlib.h>
import "C"

import (
	"unsafe"
	"github.com/mon-ius/dbox/src/libdbox"
)

//export PrintDebug
func PrintDebug(message *C.char) {
    goMsg := C.GoString(message)
    libdbox.PrintDebugMessage(goMsg)
}

//export FreeString
func FreeString(str *C.char) {
    libdbox.FreeStringPtr(unsafe.Pointer(str))
}