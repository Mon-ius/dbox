package main

// #include <stdlib.h>
import "C"
import (
	"github.com/mon-ius/dbox/src/libdbox"
)

//export Add
func Add(a, b C.int) C.int {
    return C.int(libdbox.Add(int(a), int(b)))
}

//export Multiply
func Multiply(a, b C.int) C.int {
    return C.int(libdbox.Multiply(int(a), int(b)))
}

//export HelloWorld
func HelloWorld() *C.char {
    return C.CString(libdbox.HelloWorldString())
}

//export Base64Decode
func Base64Decode(encodedStr *C.char) *C.char {
    goStr := C.GoString(encodedStr)
    result, err := libdbox.Base64DecodeString(goStr)
    if err != nil {
        return C.CString("Error: " + err.Error())
    }
    return C.CString(result)
}

//export enforce_binding
func enforce_binding() {}

func main() {}