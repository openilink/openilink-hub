package mockserver

import (
	"crypto/rand"
	"encoding/hex"

	ilink "github.com/openilink/openilink-sdk-go"
)

// generateAESKey returns a random 16-byte AES key and its hex encoding.
func generateAESKey() (raw []byte, hexKey string) {
	raw = make([]byte, 16)
	_, _ = rand.Read(raw)
	return raw, hex.EncodeToString(raw)
}

// generateEQP returns a random mock encrypted query parameter.
func generateEQP() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "mock-eqp-" + hex.EncodeToString(b)
}

// encryptMedia encrypts plaintext using AES-128-ECB with PKCS7 padding.
func encryptMedia(plaintext, key []byte) ([]byte, error) {
	return ilink.EncryptAESECB(plaintext, key)
}

// decryptMedia decrypts ciphertext using AES-128-ECB with PKCS7 padding.
func decryptMedia(ciphertext, key []byte) ([]byte, error) {
	return ilink.DecryptAESECB(ciphertext, key)
}

// parseAESKey decodes a base64-encoded AES key.
func parseAESKey(aesKeyBase64 string) ([]byte, error) {
	return ilink.ParseAESKey(aesKeyBase64)
}
