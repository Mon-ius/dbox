package libdbox

import (
	"encoding/base64"
)

func Add(a, b int) int {
    return a + b
}

func Multiply(a, b int) int {
    return a * b
}

func Base64DecodeString(encodedStr string) (string, error) {
    decodedBytes, err := base64.StdEncoding.DecodeString(encodedStr)
    if err != nil {
        return "", err
    }
    return string(decodedBytes), nil
}

func HelloWorldString() string {
    return "Hello from Go shared library!"
}