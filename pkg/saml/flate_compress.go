package saml

import (
	"bytes"
	"compress/flate"
	"io"
)

// flateCompress performs raw DEFLATE compression (RFC 1951) without a zlib
// wrapper. This is the encoding required by SAML HTTP-Redirect binding
// (SAMLBind section 3.4.4.1).
func flateCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// flateDecompress reverses flateCompress.
func flateDecompress(data []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	return io.ReadAll(r)
}
