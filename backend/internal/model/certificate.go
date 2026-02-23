package model

import (
	"time"

	"github.com/google/uuid"
)

type Certificate struct {
	ID                uuid.UUID  `db:"id"                 json:"id"`
	StudentID         uuid.UUID  `db:"student_id"         json:"student_id"`
	CertificateNumber string     `db:"certificate_number" json:"certificate_number"`
	IssuedAt          time.Time  `db:"issued_at"          json:"issued_at"`
	IssuedBy          *uuid.UUID `db:"issued_by"          json:"issued_by"`
	ValidUntil        *time.Time `db:"valid_until"        json:"valid_until"`
	QRToken           string     `db:"qr_token"           json:"qr_token"`
	PDFURL            *string    `db:"pdf_url"            json:"pdf_url"`
	Status            string     `db:"status"             json:"status"` // active | revoked
	Notes             string     `db:"notes"              json:"notes"`
	CreatedAt         time.Time  `db:"created_at"         json:"created_at"`

	// Join fields
	StudentName *string `db:"student_name" json:"student_name,omitempty"`
	StudentNISN *string `db:"student_nisn" json:"student_nisn,omitempty"`
	IssuedByName *string `db:"issued_by_name" json:"issued_by_name,omitempty"`
}

type CertificateDetail struct {
	Certificate
	Student      *Student                 `json:"student"`
	Achievements []AchievementWithAttachments `json:"achievements"`
}

type CreateCertificateRequest struct {
	StudentID      string   `json:"student_id"`
	AchievementIDs []string `json:"achievement_ids"` // prestasi yang dimasukkan ke surat
	ValidUntil     string   `json:"valid_until"`     // format: YYYY-MM-DD, opsional
	Notes          string   `json:"notes"`
}

type CertificateFilter struct {
	StudentID string
	Status    string
	Page      int
	PerPage   int
}

// VerifyResponse untuk endpoint publik verifikasi QR
type VerifyResponse struct {
	IsValid       bool             `json:"is_valid"`
	Certificate   *Certificate     `json:"certificate,omitempty"`
	Student       *Student         `json:"student,omitempty"`
	Achievements  []Achievement    `json:"achievements,omitempty"`
	Message       string           `json:"message"`
}