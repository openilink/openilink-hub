package ilink

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	ilink "github.com/openilink/openilink-sdk-go"
	"github.com/youthlin/silk"
)

func TestSilkEncodeDecodeRoundTrip(t *testing.T) {
	// Generate a simple 16-bit PCM at 24kHz, 1 second of silence
	sampleRate := 24000
	duration := 1 // seconds
	numSamples := sampleRate * duration
	pcm := make([]byte, numSamples*2) // 16-bit = 2 bytes per sample

	// Encode PCM → SILK without STX
	silkNoStx, err := silk.Encode(bytes.NewReader(pcm), silk.SampleRate(sampleRate))
	if err != nil {
		t.Fatalf("encode without stx: %v", err)
	}
	t.Logf("SILK without STX: %d bytes, header: %x", len(silkNoStx), silkNoStx[:min(10, len(silkNoStx))])

	// Encode PCM → SILK with STX (WeChat compatible)
	silkWithStx, err := silk.Encode(bytes.NewReader(pcm), silk.SampleRate(sampleRate), silk.Stx(true))
	if err != nil {
		t.Fatalf("encode with stx: %v", err)
	}
	t.Logf("SILK with STX: %d bytes, header: %x", len(silkWithStx), silkWithStx[:min(10, len(silkWithStx))])

	// Verify STX version starts with 0x02
	if silkWithStx[0] != 0x02 {
		t.Errorf("STX version should start with 0x02, got 0x%02x", silkWithStx[0])
	}

	// Verify non-STX version starts with SILK header "#!SILK_V3"
	silkHeader := "#!SILK_V3"
	if !bytes.HasPrefix(silkNoStx, []byte(silkHeader)) {
		t.Errorf("non-STX should start with %q, got %x", silkHeader, silkNoStx[:9])
	}

	// Decode STX version back to PCM
	decoded, err := silk.Decode(bytes.NewReader(silkWithStx), silk.WithSampleRate(sampleRate))
	if err != nil {
		t.Fatalf("decode stx silk: %v", err)
	}
	t.Logf("decoded PCM: %d bytes (original: %d)", len(decoded), len(pcm))
}

// TestSilkFullRoundTrip simulates the exact send flow:
// 1. Start with SILK data (as received from WeChat)
// 2. Decode SILK → WAV (what we store/serve to user)
// 3. Parse WAV back to PCM (what sendVoice does)
// 4. Encode PCM → SILK with STX (what we upload to CDN)
// 5. Verify the result is valid SILK
func TestSilkFullRoundTrip(t *testing.T) {
	sampleRate := 24000

	// Step 1: Generate original SILK (simulating what WeChat sends)
	// First create PCM, encode to SILK with STX (as WeChat does)
	originalPCM := generateTone(440, sampleRate, 1) // 1 second of 440Hz
	t.Logf("original PCM: %d bytes", len(originalPCM))

	originalSilk, err := silk.Encode(bytes.NewReader(originalPCM), silk.SampleRate(sampleRate), silk.Stx(true))
	if err != nil {
		t.Fatalf("create original silk: %v", err)
	}
	t.Logf("original SILK: %d bytes, header: %x", len(originalSilk), originalSilk[:min(10, len(originalSilk))])

	// Step 2: Decode SILK → PCM (what our DownloadVoice does)
	decodedPCM, err := silk.Decode(bytes.NewReader(originalSilk), silk.WithSampleRate(sampleRate))
	if err != nil {
		t.Fatalf("decode silk: %v", err)
	}
	t.Logf("decoded PCM: %d bytes", len(decodedPCM))

	// Step 3: Wrap PCM as WAV (what we serve to the user)
	wav := buildWAV(decodedPCM, sampleRate, 1, 16)
	t.Logf("WAV file: %d bytes", len(wav))

	// Step 4: Parse WAV back (what sendVoice does)
	info, err := parseWAV(wav)
	if err != nil {
		t.Fatalf("parse wav: %v", err)
	}
	t.Logf("parsed WAV: rate=%d, channels=%d, bits=%d, pcm=%d bytes",
		info.SampleRate, info.Channels, info.BitsPerSample, len(info.PCMData))

	if info.SampleRate != sampleRate {
		t.Errorf("sample rate: got %d, want %d", info.SampleRate, sampleRate)
	}
	if info.Channels != 1 {
		t.Errorf("channels: got %d, want 1", info.Channels)
	}
	if info.BitsPerSample != 16 {
		t.Errorf("bits: got %d, want 16", info.BitsPerSample)
	}

	// Step 5: Re-encode to SILK with STX (what sendVoice does)
	reEncodedSilk, err := silk.Encode(bytes.NewReader(info.PCMData), silk.SampleRate(info.SampleRate), silk.Stx(true))
	if err != nil {
		t.Fatalf("re-encode silk: %v", err)
	}
	t.Logf("re-encoded SILK: %d bytes, header: %x", len(reEncodedSilk), reEncodedSilk[:min(10, len(reEncodedSilk))])

	// Verify valid SILK with STX
	if reEncodedSilk[0] != 0x02 {
		t.Errorf("re-encoded should start with 0x02, got 0x%02x", reEncodedSilk[0])
	}
	if !bytes.Contains(reEncodedSilk, []byte("#!SILK_V3")) {
		t.Error("re-encoded missing SILK_V3 header")
	}

	// Step 6: Verify we can decode the re-encoded SILK
	finalPCM, err := silk.Decode(bytes.NewReader(reEncodedSilk), silk.WithSampleRate(sampleRate))
	if err != nil {
		t.Fatalf("decode re-encoded silk: %v", err)
	}
	t.Logf("final PCM: %d bytes", len(finalPCM))

	if len(finalPCM) == 0 {
		t.Error("final PCM is empty")
	}
}

func TestStereoToMono(t *testing.T) {
	// Create stereo PCM: L=1000, R=-1000 → mono should be 0
	stereo := make([]byte, 8) // 2 samples × 2 channels × 2 bytes
	neg1000 := int16(-1000)
	binary.LittleEndian.PutUint16(stereo[0:], 1000)            // L
	binary.LittleEndian.PutUint16(stereo[2:], uint16(neg1000)) // R
	binary.LittleEndian.PutUint16(stereo[4:], 500)             // L
	binary.LittleEndian.PutUint16(stereo[6:], 500)             // R

	mono := stereoToMono(stereo)
	if len(mono) != 4 {
		t.Fatalf("mono length: got %d, want 4", len(mono))
	}

	s1 := int16(binary.LittleEndian.Uint16(mono[0:]))
	s2 := int16(binary.LittleEndian.Uint16(mono[2:]))
	t.Logf("sample1=%d (expect ~0), sample2=%d (expect 500)", s1, s2)

	if s1 != 0 {
		t.Errorf("sample1: got %d, want 0", s1)
	}
	if s2 != 500 {
		t.Errorf("sample2: got %d, want 500", s2)
	}
}

func TestParseWAV(t *testing.T) {
	pcm := make([]byte, 1000)
	wav := buildWAV(pcm, 48000, 2, 16)

	info, err := parseWAV(wav)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if info.SampleRate != 48000 {
		t.Errorf("rate: %d", info.SampleRate)
	}
	if info.Channels != 2 {
		t.Errorf("channels: %d", info.Channels)
	}
	if len(info.PCMData) != 1000 {
		t.Errorf("pcm: %d bytes", len(info.PCMData))
	}
}

// --- helpers ---

func generateTone(freqHz, sampleRate, durationSec int) []byte {
	n := sampleRate * durationSec
	pcm := make([]byte, n*2)
	for i := 0; i < n; i++ {
		// Simple sine approximation using integer math
		phase := (i * freqHz * 4) / sampleRate
		var val int16
		switch phase % 4 {
		case 0:
			val = 0
		case 1:
			val = 16000
		case 2:
			val = 0
		case 3:
			val = -16000
		}
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(val))
	}
	return pcm
}

func buildWAV(pcm []byte, sampleRate, channels, bitsPerSample int) []byte {
	blockAlign := channels * bitsPerSample / 8
	byteRate := sampleRate * blockAlign
	dataSize := len(pcm)
	fileSize := 36 + dataSize

	buf := make([]byte, 44+dataSize)
	copy(buf[0:], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:], uint32(fileSize))
	copy(buf[8:], "WAVE")
	copy(buf[12:], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:], 16) // fmt chunk size
	binary.LittleEndian.PutUint16(buf[20:], 1)  // PCM
	binary.LittleEndian.PutUint16(buf[22:], uint16(channels))
	binary.LittleEndian.PutUint32(buf[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:], uint32(byteRate))
	binary.LittleEndian.PutUint16(buf[32:], uint16(blockAlign))
	binary.LittleEndian.PutUint16(buf[34:], uint16(bitsPerSample))
	copy(buf[36:], "data")
	binary.LittleEndian.PutUint32(buf[40:], uint32(dataSize))
	copy(buf[44:], pcm)
	return buf
}

func TestConvertItemFallbackVoiceByPayload(t *testing.T) {
	item := ilink.MessageItem{
		Type: 99,
		VoiceItem: &ilink.VoiceItem{
			Text: "fallback voice",
			Media: &ilink.CDNMedia{
				EncryptQueryParam: "eqp",
				AESKey:            "aes",
			},
			SampleRate:    16000,
			BitsPerSample: 16,
			EncodeType:    6,
			PlayTime:      2,
		},
	}

	got := convertItem(item)
	if got == nil {
		t.Fatal("convertItem returned nil")
	}
	if got.Type != "voice" {
		t.Fatalf("type = %q, want voice", got.Type)
	}
	if got.Text != "fallback voice" {
		t.Fatalf("text = %q", got.Text)
	}
	if got.Media == nil {
		t.Fatal("media is nil")
	}
	if got.Media.SampleRate != 16000 || got.Media.EncodeType != 6 {
		t.Fatalf("media = %+v", got.Media)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	// Suppress unused import
	_ = fmt.Sprintf
}
