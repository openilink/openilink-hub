package auth

import (
	"bytes"
	"testing"
)

func TestCredentialIDEncodingRoundTrip(t *testing.T) {
	original := []byte{0x00, 0x01, 0x02, 0x7f, 0x80, 0xfe, 0xff}
	encoded := EncodeCredentialID(original)
	decoded := DecodeCredentialID(encoded)

	if !bytes.Equal(decoded, original) {
		t.Fatalf("decoded credential id mismatch: got %v want %v", decoded, original)
	}
}

func TestCredentialIDDecodeLegacyFallback(t *testing.T) {
	legacy := "cred-id-2"
	decoded := DecodeCredentialID(legacy)

	if !bytes.Equal(decoded, []byte(legacy)) {
		t.Fatalf("legacy credential id mismatch: got %v want %v", decoded, []byte(legacy))
	}
}
