package main

import (
	"crypto/rand"
	"encoding/hex"
)

func generateShortCode() (string, error) {
	bytes := make([]byte, 4)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
