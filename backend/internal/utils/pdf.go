package utils

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type CertificatePDFData struct {
	CertificateNumber string
	IssuedAt          time.Time
	ValidUntil        *time.Time
	SchoolName        string
	SchoolAddress     string
	Student           PDFStudent
	Achievements      []PDFAchievement
	QRCodePNG         []byte // QR code sebagai bytes PNG
	HeadmasterName    string
	HeadmasterNIP     string
}

type PDFStudent struct {
	FullName   string
	NISN       string
	BirthPlace string
	BirthDate  string
	Class      string
}

type PDFAchievement struct {
	No              int
	CompetitionName string
	Organizer       string
	Category        string
	Rank            string
	Level           string
	Year            int
}

func GenerateCertificatePDF(data CertificatePDFData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// ─────────────────────────────────────────
	// HEADER - Kop Surat
	// ─────────────────────────────────────────
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(0, 51, 102)
	pdf.CellFormat(0, 8, data.SchoolName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 5, data.SchoolAddress, "", 1, "C", false, 0, "")

	// Garis pembatas
	pdf.SetDrawColor(0, 51, 102)
	pdf.SetLineWidth(0.8)
	pdf.Line(20, pdf.GetY()+3, 190, pdf.GetY()+3)
	pdf.Ln(8)

	// ─────────────────────────────────────────
	// JUDUL SURAT
	// ─────────────────────────────────────────
	pdf.SetFont("Arial", "B", 13)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 8, "SURAT KETERANGAN PRESTASI", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 5, fmt.Sprintf("Nomor: %s", data.CertificateNumber), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// ─────────────────────────────────────────
	// PEMBUKA
	// ─────────────────────────────────────────
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(0, 6,
		"Yang bertanda tangan di bawah ini, Kepala Sekolah menerangkan bahwa siswa berikut:",
		"", "L", false)
	pdf.Ln(3)

	// ─────────────────────────────────────────
	// DATA SISWA
	// ─────────────────────────────────────────
	colLabel := 50.0
	colValue := 120.0

	dataRows := [][]string{
		{"Nama Lengkap", data.Student.FullName},
		{"NISN", data.Student.NISN},
		{"Tempat, Tanggal Lahir", fmt.Sprintf("%s, %s", data.Student.BirthPlace, data.Student.BirthDate)},
		{"Kelas", data.Student.Class},
	}

	pdf.SetFont("Arial", "", 10)
	for _, row := range dataRows {
		pdf.CellFormat(colLabel, 6, row[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(5, 6, ":", "", 0, "C", false, 0, "")
		pdf.CellFormat(colValue, 6, row[1], "", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	// ─────────────────────────────────────────
	// KALIMAT TENGAH
	// ─────────────────────────────────────────
	pdf.MultiCell(0, 6,
		"Telah meraih prestasi sebagai berikut:",
		"", "L", false)
	pdf.Ln(3)

	// ─────────────────────────────────────────
	// TABEL PRESTASI
	// ─────────────────────────────────────────
	// Header tabel
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(0, 51, 102)
	pdf.SetTextColor(255, 255, 255)

	headers := []string{"No", "Nama Lomba", "Penyelenggara", "Kategori", "Juara", "Tingkat", "Tahun"}
	widths := []float64{8, 45, 35, 22, 18, 22, 15}

	for i, h := range headers {
		pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Isi tabel
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(0, 0, 0)

	for i, a := range data.Achievements {
		fillColor := i%2 == 0
		if fillColor {
			pdf.SetFillColor(240, 245, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.CellFormat(widths[0], 6, fmt.Sprintf("%d", a.No), "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(widths[1], 6, truncate(a.CompetitionName, 30), "1", 0, "L", fillColor, 0, "")
		pdf.CellFormat(widths[2], 6, truncate(a.Organizer, 22), "1", 0, "L", fillColor, 0, "")
		pdf.CellFormat(widths[3], 6, truncate(a.Category, 14), "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(widths[4], 6, a.Rank, "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(widths[5], 6, truncate(a.Level, 14), "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(widths[6], 6, fmt.Sprintf("%d", a.Year), "1", 0, "C", fillColor, 0, "")
		pdf.Ln(-1)
	}
	pdf.Ln(5)

	// ─────────────────────────────────────────
	// PENUTUP
	// ─────────────────────────────────────────
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(0, 6,
		"Surat keterangan ini dibuat dengan sebenarnya untuk dapat dipergunakan sebagaimana mestinya.",
		"", "L", false)
	pdf.Ln(5)

	// ─────────────────────────────────────────
	// TANGGAL & TANDA TANGAN
	// ─────────────────────────────────────────
	bulan := [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

	issuedDate := fmt.Sprintf("%d %s %d",
		data.IssuedAt.Day(),
		bulan[data.IssuedAt.Month()],
		data.IssuedAt.Year(),
	)

	// Kolom kiri: QR code, Kolom kanan: TTD
	currentY := pdf.GetY()
	pageWidth := 170.0

	// QR Code (kiri)
	if len(data.QRCodePNG) > 0 {
		pdf.SetFont("Arial", "", 8)
		pdf.SetXY(20, currentY)
		pdf.CellFormat(40, 5, "Scan untuk verifikasi:", "", 1, "L", false, 0, "")

		qrReader := bytes.NewReader(data.QRCodePNG)
		pdf.RegisterImageOptionsReader("qrcode", gofpdf.ImageOptions{ImageType: "PNG"}, qrReader)
		pdf.ImageOptions("qrcode", 20, currentY+6, 35, 35, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	// TTD (kanan)
	signX := 20 + pageWidth - 65
	pdf.SetXY(signX, currentY)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(65, 5, fmt.Sprintf("Diterbitkan, %s", issuedDate), "", 1, "C", false, 0, "")
	pdf.SetX(signX)
	pdf.CellFormat(65, 5, "Kepala Sekolah,", "", 1, "C", false, 0, "")
	pdf.Ln(18) // ruang tanda tangan
	pdf.SetX(signX)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(65, 5, data.HeadmasterName, "", 1, "C", false, 0, "")
	if data.HeadmasterNIP != "" {
		pdf.SetX(signX)
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(65, 5, fmt.Sprintf("NIP. %s", data.HeadmasterNIP), "", 1, "C", false, 0, "")
	}

	// ─────────────────────────────────────────
	// FOOTER
	// ─────────────────────────────────────────
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 7)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(0, 5,
		fmt.Sprintf("Dokumen ini diterbitkan secara digital pada %s | Verifikasi keaslian dokumen dengan scan QR Code",
			data.IssuedAt.Format("02/01/2006 15:04")),
		"", 1, "C", false, 0, "")

	// Output ke bytes
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gagal generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}