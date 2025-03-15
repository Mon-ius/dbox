//go:build with_debug

package libdbox

import (
    "fmt"
    "unsafe"
)

// #include <stdlib.h>
import "C"

func PrintDebugMessage(message string) {
    fmt.Printf("DEBUG: %s\n", message)
}

func FreeStringPtr(ptr unsafe.Pointer) {
    C.free(ptr)
}