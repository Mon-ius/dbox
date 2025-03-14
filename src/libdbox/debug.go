//go:build with_debug

package libdbox

// #include <stdlib.h>
import "C"
import (
	"fmt"
	"unsafe"
)

//export PrintDebug
func PrintDebug(message *C.char) {
	fmt.Printf("DEBUG: %s\n", C.GoString(message))
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}