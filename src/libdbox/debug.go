//go:build with_debug

package libdbox

import (
	"fmt"
	"unsafe"
)

// #include <stdlib.h>
import "C"

//export PrintDebug
func PrintDebug(message *C.char) {
	fmt.Printf("DEBUG: %s\n", C.GoString(message))
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}