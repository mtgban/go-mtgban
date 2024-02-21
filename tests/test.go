package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

func generateSHA256ID(data string) string {
	hasher := sha256.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
func generateBase64SHA256ID(data string) string {
	hasher := sha256.New()
	hasher.Write([]byte(data))
	hashDigest := hasher.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(hashDigest)
}

func main() {
	test_inputs := []string{"f6555d1f-d4cf-41f7-99d3-88fd53e75457-NM-FOIL-ENGLISH"}
	for _, data := range test_inputs {
		sha256ID := generateSHA256ID(data)
		base64SHA256ID := generateBase64SHA256ID(data)
		fmt.Println("Input:", data)
		fmt.Println("SHA-256 ID:", sha256ID)
		fmt.Println("Base64 SHA-256 ID:", base64SHA256ID)
		fmt.Println("-----------------------")
	}
}
