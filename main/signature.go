package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/fatih/color"
)

var privateKey *ecdsa.PrivateKey = nil
var publicKey *ecdsa.PublicKey = nil

var debug_signature bool = false

func GenKeys() {
	if debug_signature {
		fmt.Println("[GenKeys] Generating Keys")
	}

	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		if debug_signature {
			color.Red("[GenKeys] Error while generating private key : %s\n", err.Error())
		}
		return
	}
	privateKey = sk

	pk, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		if debug_signature {
			color.Red("[GenKeys] Error while converting public key\n")
		}
		privateKey = nil
		return
	}
	publicKey = pk

	if debug_signature {
		fmt.Println("[GenKeys] Done")
	}
}

// Pour parser une clé publique représentée comme une chaîne de 64 octets :
func parsePubKey(data []byte) *ecdsa.PublicKey {
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
		fmt.Println("[computeSignature] Signing data")
	}

	hashed := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashed[:])
	if err != nil {
		if debug_signature {
			color.Red("[computeSignature] Error while computing signature : %s\n", err.Error())
		}
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	if debug_signature {
		fmt.Println("[computeSignature] Done")
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
		if debug_signature {
			fmt.Println("[VerifySignature] Key is nil or full of zeros, skipping verification")
		}
		return true
	}

	// Here, the peer is supposed to have a key
	// So, if there no signature, we should throw the packet out
	if len(signature) == 0 {
		if debug_signature {
			color.Red("[VerifySignature] Peer has a key but message is not signed\n")
		}
		return false
	}

	pk := parsePubKey(key)
	var r, s big.Int
	r.SetBytes(signature[:32])
	s.SetBytes(signature[32:])
	hashed := sha256.Sum256(data)
	return ecdsa.Verify(pk, hashed[:], &r, &s)
}
