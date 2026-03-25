package mockserver

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key, _ := generateAESKey()
	plaintext := []byte("hello, iLink mock server!")

	ciphertext, err := encryptMedia(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := decryptMedia(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1, _ := generateAESKey()
	key2, _ := generateAESKey()
	plaintext := []byte("secret data")

	ciphertext, err := encryptMedia(plaintext, key1)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := decryptMedia(ciphertext, key2)
	// Decryption with wrong key should either error or produce different data.
	if err == nil && bytes.Equal(got, plaintext) {
		t.Fatal("decryption with wrong key should not return original plaintext")
	}
}

func TestGenerateAESKey(t *testing.T) {
	raw, hexKey := generateAESKey()
	if len(raw) != 16 {
		t.Fatalf("key length = %d, want 16", len(raw))
	}
	decoded, err := hex.DecodeString(hexKey)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	if !bytes.Equal(decoded, raw) {
		t.Fatal("hex encoding does not match raw key")
	}
}

func TestGenerateEQP(t *testing.T) {
	eqp := generateEQP()
	if len(eqp) < 10 {
		t.Fatal("EQP too short")
	}
	if eqp[:9] != "mock-eqp-" {
		t.Fatalf("EQP prefix = %q, want 'mock-eqp-'", eqp[:9])
	}
}

func TestParseAESKey_StdBase64(t *testing.T) {
	raw, _ := generateAESKey()
	encoded := base64.StdEncoding.EncodeToString(raw)
	got, err := parseAESKey(encoded)
	if err != nil {
		t.Fatalf("parseAESKey(std): %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatal("parsed key does not match original")
	}
}

func TestParseAESKey_URLBase64(t *testing.T) {
	raw, _ := generateAESKey()
	encoded := base64.URLEncoding.EncodeToString(raw)
	got, err := parseAESKey(encoded)
	if err != nil {
		t.Fatalf("parseAESKey(url): %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatal("parsed key does not match original")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key, _ := generateAESKey()
	ciphertext, err := encryptMedia([]byte{}, key)
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}
	got, err := decryptMedia(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty plaintext, got %d bytes", len(got))
	}
}
