package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/ahmadqo/digital-achievement-ledger/internal/repository"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
)

var (
	ErrCertificateNotFound = errors.New("sertifikat tidak ditemukan")
	ErrCertificateRevoked  = errors.New("sertifikat telah dicabut")
)

type CertificateService interface {
	GetAll(ctx context.Context, filter model.CertificateFilter) ([]*model.Certificate, *response.Pagination, error)
	GetByID(ctx context.Context, id string) (*model.CertificateDetail, error)
	Create(ctx context.Context, req model.CreateCertificateRequest, issuedBy string) (*model.CertificateDetail, error)
	Revoke(ctx context.Context, id string) error
	Verify(ctx context.Context, token string) (*model.VerifyResponse, error)
	DownloadPDF(ctx context.Context, id string) ([]byte, string, error)
}

type certificateService struct {
	repo        repository.CertificateRepository
	studentRepo repository.StudentRepository
	achRepo     repository.AchievementRepository
	storage     *utils.StorageService
}

func NewCertificateService(
	repo repository.CertificateRepository,
	studentRepo repository.StudentRepository,
	achRepo repository.AchievementRepository,
	storage *utils.StorageService,
) CertificateService {
	return &certificateService{
		repo: repo, studentRepo: studentRepo,
		achRepo: achRepo, storage: storage,
	}
}

func (s *certificateService) GetAll(ctx context.Context, filter model.CertificateFilter) ([]*model.Certificate, *response.Pagination, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 10
	}

	certs, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	totalPages := int(total) / filter.PerPage
	if int(total)%filter.PerPage > 0 {
		totalPages++
	}

	return certs, &response.Pagination{
		Page: filter.Page, PerPage: filter.PerPage,
		TotalItems: total, TotalPages: totalPages,
	}, nil
}

func (s *certificateService) GetByID(ctx context.Context, id string) (*model.CertificateDetail, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("ID tidak valid")
	}

	detail, err := s.repo.FindByIDWithDetail(ctx, uid)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrCertificateNotFound
	}

	return detail, nil
}

func (s *certificateService) Create(ctx context.Context, req model.CreateCertificateRequest, issuedBy string) (*model.CertificateDetail, error) {
	studentUID, err := uuid.Parse(req.StudentID)
	if err != nil {
		return nil, errors.New("student_id tidak valid")
	}

	// Validasi achievement IDs
	if len(req.AchievementIDs) == 0 {
		return nil, errors.New("minimal 1 prestasi harus dipilih")
	}

	achievementUIDs := make([]uuid.UUID, 0, len(req.AchievementIDs))
	for _, idStr := range req.AchievementIDs {
		uid, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("achievement_id tidak valid: %s", idStr)
		}
		achievementUIDs = append(achievementUIDs, uid)
	}

	// Generate nomor surat
	year := time.Now().Year()
	count, err := s.repo.CountByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	certNumber := utils.GenerateCertificateNumber(count + 1)

	// Generate QR token
	qrToken, err := utils.GenerateQRToken()
	if err != nil {
		return nil, err
	}

	issuedByUID, _ := uuid.Parse(issuedBy)

	cert := &model.Certificate{
		ID:                uuid.New(),
		StudentID:         studentUID,
		CertificateNumber: certNumber,
		IssuedAt:          time.Now(),
		IssuedBy:          &issuedByUID,
		QRToken:           qrToken,
		Status:            "active",
		Notes:             req.Notes,
	}

	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err != nil {
			return nil, errors.New("format valid_until tidak valid, gunakan YYYY-MM-DD")
		}
		cert.ValidUntil = &t
	}

	// Simpan ke DB
	if err := s.repo.Create(ctx, cert, achievementUIDs); err != nil {
		return nil, err
	}

	// Ambil detail lengkap
	detail, err := s.repo.FindByIDWithDetail(ctx, cert.ID)
	if err != nil {
		return nil, err
	}

	// Generate dan upload PDF di background
	go s.generateAndUploadPDF(context.Background(), detail)

	return detail, nil
}

func (s *certificateService) generateAndUploadPDF(ctx context.Context, detail *model.CertificateDetail) {
	pdfBytes, certNumber, err := s.buildPDF(detail)
	if err != nil {
		return
	}

	pdfURL, err := s.storage.UploadPDF(ctx, "certificates", pdfBytes, certNumber)
	if err != nil {
		return
	}

	s.repo.UpdatePDFURL(ctx, detail.Certificate.ID, pdfURL)
}

func (s *certificateService) DownloadPDF(ctx context.Context, id string) ([]byte, string, error) {
	detail, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}

	pdfBytes, certNumber, err := s.buildPDF(detail)
	if err != nil {
		return nil, "", err
	}

	return pdfBytes, certNumber, nil
}

func (s *certificateService) buildPDF(detail *model.CertificateDetail) ([]byte, string, error) {
	// Build achievements untuk PDF
	pdfAchievements := make([]utils.PDFAchievement, len(detail.Achievements))
	for i, a := range detail.Achievements {
		category := ""
		if a.CategoryName != nil {
			category = *a.CategoryName
		}
		level := ""
		if a.LevelName != nil {
			level = *a.LevelName
		}
		pdfAchievements[i] = utils.PDFAchievement{
			No:              i + 1,
			CompetitionName: a.CompetitionName,
			Organizer:       a.Organizer,
			Category:        category,
			Rank:            a.Rank,
			Level:           level,
			Year:            a.Year,
		}
	}

	// Build student data untuk PDF
	birthDate := ""
	if detail.Student.BirthDate != nil {
		birthDate = detail.Student.BirthDate.Format("02 January 2006")
	}

	// Generate QR code
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}
	verifyURL := fmt.Sprintf("%s/api/v1/verify/%s", appURL, detail.Certificate.QRToken)

	qrPNG, _ := utils.GenerateQRCodePNG(verifyURL, 150)

	// Data sekolah dari env
	schoolName := os.Getenv("SCHOOL_NAME")
	if schoolName == "" {
		schoolName = "SMA Negeri 1"
	}
	schoolAddress := os.Getenv("SCHOOL_ADDRESS")
	if schoolAddress == "" {
		schoolAddress = "Jl. Pendidikan No. 1"
	}
	headmasterName := os.Getenv("HEADMASTER_NAME")
	if headmasterName == "" {
		headmasterName = "Kepala Sekolah"
	}
	headmasterNIP := os.Getenv("HEADMASTER_NIP")

	pdfData := utils.CertificatePDFData{
		CertificateNumber: detail.Certificate.CertificateNumber,
		IssuedAt:          detail.Certificate.IssuedAt,
		ValidUntil:        detail.Certificate.ValidUntil,
		SchoolName:        schoolName,
		SchoolAddress:     schoolAddress,
		Student: utils.PDFStudent{
			FullName:   detail.Student.FullName,
			NISN:       detail.Student.NISN,
			BirthPlace: detail.Student.BirthPlace,
			BirthDate:  birthDate,
			Class:      detail.Student.Class,
		},
		Achievements:   pdfAchievements,
		QRCodePNG:      qrPNG,
		HeadmasterName: headmasterName,
		HeadmasterNIP:  headmasterNIP,
	}

	pdfBytes, err := utils.GenerateCertificatePDF(pdfData)
	return pdfBytes, detail.Certificate.CertificateNumber, err
}

func (s *certificateService) Revoke(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.New("ID tidak valid")
	}

	cert, err := s.repo.FindByID(ctx, uid)
	if err != nil {
		return err
	}
	if cert == nil {
		return ErrCertificateNotFound
	}
	if cert.Status == "revoked" {
		return errors.New("sertifikat sudah dicabut sebelumnya")
	}

	return s.repo.Revoke(ctx, uid)
}

func (s *certificateService) Verify(ctx context.Context, token string) (*model.VerifyResponse, error) {
	cert, err := s.repo.FindByQRToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if cert == nil {
		return &model.VerifyResponse{
			IsValid: false,
			Message: "Sertifikat tidak ditemukan. Dokumen ini mungkin tidak sah.",
		}, nil
	}

	if cert.Status == "revoked" {
		return &model.VerifyResponse{
			IsValid:     false,
			Certificate: cert,
			Message:     "Sertifikat ini telah dicabut dan tidak berlaku.",
		}, nil
	}

	// Ambil detail lengkap
	uid := cert.ID
	detail, err := s.repo.FindByIDWithDetail(ctx, uid)
	if err != nil {
		return nil, err
	}

	achievements := make([]model.Achievement, len(detail.Achievements))
	for i, a := range detail.Achievements {
		achievements[i] = a.Achievement
	}

	return &model.VerifyResponse{
		IsValid:      true,
		Certificate:  cert,
		Student:      detail.Student,
		Achievements: achievements,
		Message:      "Sertifikat valid dan sah dikeluarkan oleh sekolah.",
	}, nil
}