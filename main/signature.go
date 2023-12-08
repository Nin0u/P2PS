package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

var privateKey *ecdsa.PrivateKey = nil
var publicKey *ecdsa.PublicKey = nil

var debug_signature bool = false

func GenKeys() {
	// Generation
	if debug_signature {
		fmt.Println("[GenKeys] Generating Keys")
	}
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		if debug_signature {
			fmt.Println("[GenKeys] Error while generating private key :", err)
		}
	}
	privateKey = sk

	pk, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		if debug_signature {
			fmt.Println("[GenKeys] Error while converting public key :", publicKey)
		}
	}

	publicKey = pk

	if debug_signature {
		fmt.Println("[GenKeys] Key generated :", privateKey, publicKey)
	}
}

// Pour parser une clé publique représentée comme une chaîne de 64 octets :
func parsePubKey(data []byte) *ecdsa.PublicKey {
	if debug_signature {
		fmt.Printf("[parsePubKey] Parsing : %x\n", data)
	}
	var x, y big.Int
	x.SetBytes(data[:32])
	y.SetBytes(data[32:])
	pk := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     &x,
		Y:     &y,
	}

	return pk
}

// Pour calculer la signature d'un message
func computeSignature(data []byte) []byte {
	if debug_signature {
		fmt.Printf("[computeSignature] Signing data = %x\n", data)
	}

	hashed := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashed[:])
	if err != nil {
		if debug_signature {
			fmt.Println("[computeSignature] Error while computing signature of a message", err)
		}
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	if debug_signature {
		fmt.Printf("[computeSignature] Sign result :%x\n", signature)
	}

	return signature
}

func isFullOfZeros(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != 0 {
			return false
		}
	}
	return true
}

func VerifySignature(key []byte, data []byte, signature []byte) bool {
	// Skip checking if no key found
	if key == nil || isFullOfZeros(key) {
		return true
	}

	pk := parsePubKey(key)
	var r, s big.Int
	r.SetBytes(signature[:32])
	s.SetBytes(signature[32:])
	hashed := sha256.Sum256(data)
	return ecdsa.Verify(pk, hashed[:], &r, &s)
}
