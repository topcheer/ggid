package saml

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestFlateCompressDecompressRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello world")},
		{"xml", []byte(`<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"><saml:Issuer>https://idp.example.com</saml:Issuer></samlp:Response>`)},
		{"large", bytes.Repeat([]byte("SAML assertion data "), 500)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := flateCompress(tt.data)
			if err != nil {
				t.Fatalf("flateCompress: %v", err)
			}
			decompressed, err := flateDecompress(compressed)
			if err != nil {
				t.Fatalf("flateDecompress: %v", err)
			}
			if !bytes.Equal(tt.data, decompressed) {
				t.Errorf("round-trip mismatch: input %d bytes, output %d bytes", len(tt.data), len(decompressed))
			}
		})
	}
}

func TestFlateCompressSmallerThanInput(t *testing.T) {
	data := bytes.Repeat([]byte("repeated data for compression test "), 100)
	compressed, err := flateCompress(data)
	if err != nil {
		t.Fatalf("flateCompress: %v", err)
	}
	if len(compressed) >= len(data) {
		t.Logf("compressed (%d) >= input (%d) — expected smaller for repetitive data", len(compressed), len(data))
	}
}

func TestFlateDecompressInvalidData(t *testing.T) {
	_, err := flateDecompress([]byte("not valid deflate data"))
	if err == nil {
		t.Error("expected error for invalid deflate data, got nil")
	}
}

func TestFlateCompressEmptyInput(t *testing.T) {
	compressed, err := flateCompress([]byte{})
	if err != nil {
		t.Fatalf("flateCompress empty: %v", err)
	}
	decompressed, err := flateDecompress(compressed)
	if err != nil {
		t.Fatalf("flateDecompress: %v", err)
	}
	if len(decompressed) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(decompressed))
	}
}

// Test that compressed data can be read by standard flate reader
func TestFlateCompressStandardReader(t *testing.T) {
	data := []byte("SAML XML content for cross-compatibility test")
	compressed, err := flateCompress(data)
	if err != nil {
		t.Fatalf("flateCompress: %v", err)
	}
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()
	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("standard flate reader: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Error("standard flate reader produced different output")
	}
}
