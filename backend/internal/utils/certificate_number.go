package utils

import (
	"fmt"
	"time"
)

// GenerateCertificateNumber membuat nomor surat format: 421.2/SKP/{YEAR}/{INCREMENT}
// increment didapat dari DB (total sertifikat tahun ini + 1)
func GenerateCertificateNumber(increment int) string {
	year := time.Now().Year()
	return fmt.Sprintf("421.2/SKP/%d/%04d", year, increment)
}