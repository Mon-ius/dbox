package mobile

import (
    "github.com/mon-ius/dbox/src/libdbox"
)

func Add(a, b int) int {
    return libdbox.Add(a, b)
}

func Multiply(a, b int) int {
    return libdbox.Multiply(a, b)
}

func HelloWorld() string {
    return libdbox.HelloWorldString()
}

func Base64Decode(encodedStr string) (string, error) {
    return libdbox.Base64DecodeString(encodedStr)
}