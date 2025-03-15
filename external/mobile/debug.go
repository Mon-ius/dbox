//go:build with_debug

package mobile

import (
    "unsafe",
    "github.com/mon-ius/dbox/src/libdbox"
)

func PrintDebug(message string) {
    libdbox.PrintDebugMessage(message)
}

func FreeString(ptr unsafe.Pointer) {
    libdbox.FreeStringPtr(ptr)
}