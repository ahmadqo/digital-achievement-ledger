package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRToken membuat token unik untuk QR code
func GenerateQRToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateQRCodePNG membuat QR code sebagai PNG bytes
func GenerateQRCodePNG(content string, size int) ([]byte, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("gagal generate QR code: %w", err)
	}
	return png, nil
}