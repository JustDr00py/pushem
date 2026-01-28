package main

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	// Generate keys using the library
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Private key: %s\n", priv)
	fmt.Printf("Private key length: %d\n", len(priv))
	fmt.Printf("Public key: %s\n", pub)
	fmt.Printf("Public key length: %d\n", len(pub))

	// Decode and inspect
	privBytes, err := base64.RawURLEncoding.DecodeString(priv)
	if err != nil {
		fmt.Printf("Failed to decode: %v\n", err)
		return
	}

	fmt.Printf("Decoded length: %d\n", len(privBytes))
	fmt.Printf("First bytes: %x\n", privBytes[:min(10, len(privBytes))])

	// Try to parse as EC key
	ecKey, err := x509.ParseECPrivateKey(privBytes)
	if err != nil {
		fmt.Printf("Not EC DER format: %v\n", err)
	} else {
		fmt.Printf("Successfully parsed as EC key\n")
		fmt.Printf("D value length: %d\n", len(ecKey.D.Bytes()))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
