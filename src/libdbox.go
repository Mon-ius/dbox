package libdbox

import (
	"encoding/base64"
	"fmt"
	"unsafe"
)

// #include <stdlib.h>
import "C"

//export Add
func Add(a, b int) int {
	return a + b
}

//export Multiply
func Multiply(a, b int) int {
	return a * b
}

//export HelloWorld
func HelloWorld() *C.char {
	return C.CString("Hello from Go shared library!")
}

//export Base64Decode
func Base64Decode(encodedStr *C.char) *C.char {
	goEncodedStr := C.GoString(encodedStr)
	
	decodedBytes, err := base64.StdEncoding.DecodeString(goEncodedStr)
	if err != nil {
		return C.CString("Error: " + err.Error())
	}
	
	return C.CString(string(decodedBytes))
}

//export PrintDebug
func PrintDebug(message *C.char) {
	fmt.Printf("DEBUG: %s\n", C.GoString(message))
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

//export enforce_binding
func enforce_binding() {}