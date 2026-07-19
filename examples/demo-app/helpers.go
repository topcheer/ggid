package main

import (
	"crypto/sha256"
	"os"
)

func newSHA256() interface {
	Write([]byte) (int, error)
	Sum([]byte) []byte
} {
	return sha256.New()
}

func getenvRaw(key string) string {
	return os.Getenv(key)
}
