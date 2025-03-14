//go:build with_debug

package mobile

import (
    "github.com/mon-ius/dbox/src/libdbox"
)

func PrintDebug(message string) {
    libdbox.PrintDebugMessage(message)
}