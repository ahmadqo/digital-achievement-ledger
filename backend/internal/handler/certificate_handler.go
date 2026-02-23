package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ahmadqo/digital-achievement-ledger/internal/middleware"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/service"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
	"github.com/go-chi/chi/v5"
)

type CertificateHandler struct {
	svc service.CertificateService
}

func NewCertificateHandler(svc service.CertificateService) *CertificateHandler {
	return &CertificateHandler{svc: svc}
}

// GetAll retrieves all certificates with optional filters
// @Summary      Get all certificates
// @Description  Get a paginated list of certificates
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        student_id  query    string  false  "Filter by student ID"
// @Param        status      query    string  false  "Filter by certificate status (active, revoked)"
// @Param        page        query    int     false  "Page number"
// @Param        per_page    query    int     false  "Items per page"
// @Security     BearerAuth
// @Success      200  {object}  response.PaginatedResponse
// @Failure      500  {object}  response.Response
// @Router       /certificates [get]
func (h *CertificateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.CertificateFilter{
		StudentID: q.Get("student_id"),
		Status:    q.Get("status"),
		Page:      parseIntQuery(q.Get("page"), 1),
		PerPage:   parseIntQuery(q.Get("per_page"), 10),
	}

	certs, pagination, err := h.svc.GetAll(r.Context(), filter)
	if err != nil {
		response.InternalError(w, "Gagal mengambil data sertifikat")
		return
	}

	response.Paginated(w, "Data sertifikat berhasil diambil", certs, pagination)
}

// GetByID retrieves a specific certificate
// @Summary      Get certificate by ID
// @Description  Get detailed information about a certificate
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Certificate ID"
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /certificates/{id} [get]
func (h *CertificateHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	cert, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCertificateNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal mengambil data sertifikat")
		return
	}

	response.Success(w, "Data sertifikat berhasil diambil", cert)
}

// Create issues a new certificate
// @Summary      Create a certificate
// @Description  Issue a new certificate for a student with selected achievements
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        request  body      model.CreateCertificateRequest  true  "Certificate creation request"
// @Security     BearerAuth
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Router       /certificates [post]
func (h *CertificateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCertificateRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	errs := utils.ValidationErrors{}
	if req.StudentID == "" {
		errs["student_id"] = "Student ID wajib diisi"
	}
	if len(req.AchievementIDs) == 0 {
		errs["achievement_ids"] = "Minimal 1 prestasi harus dipilih"
	}
	if errs.HasErrors() {
		response.BadRequest(w, "Validasi gagal", errs)
		return
	}

	issuedBy := middleware.GetUserIDFromContext(r.Context())
	cert, err := h.svc.Create(r.Context(), req, issuedBy)
	if err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.BadRequest(w, err.Error(), nil)
		return
	}

	response.Created(w, "Sertifikat berhasil diterbitkan", cert)
}

// Revoke invalidates a certificate
// @Summary      Revoke a certificate
// @Description  Revoke an issued certificate
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Certificate ID"
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /certificates/{id}/revoke [post]
func (h *CertificateHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Revoke(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrCertificateNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.BadRequest(w, err.Error(), nil)
		return
	}

	response.Success(w, "Sertifikat berhasil dicabut", nil)
}

// Download generates and returns the certificate PDF
// @Summary      Download certificate PDF
// @Description  Generate and download the PDF for a specific certificate
// @Tags         certificates
// @Produce      application/pdf
// @Param        id   path      string  true  "Certificate ID"
// @Security     BearerAuth
// @Success      200  {file}    file    "Certificate PDF file"
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /certificates/{id}/download [get]
func (h *CertificateHandler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	pdfBytes, certNumber, err := h.svc.DownloadPDF(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCertificateNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal generate PDF")
		return
	}

	// Set header untuk download PDF
	filename := fmt.Sprintf("SKP-%s.pdf", certNumber)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}

// Verify checks the validity of a certificate via its public token
// @Summary      Verify a certificate
// @Description  Public verify endpoint for a certificate token
// @Tags         public
// @Accept       json
// @Produce      json
// @Param        token  path      string  true  "Verification Token"
// @Success      200    {object}  response.Response
// @Failure      422    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Router       /verify/{token} [get]
func (h *CertificateHandler) Verify(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	result, err := h.svc.Verify(r.Context(), token)
	if err != nil {
		response.InternalError(w, "Gagal memverifikasi sertifikat")
		return
	}

	if !result.IsValid {
		response.JSON(w, http.StatusUnprocessableEntity, false, result.Message, result)
		return
	}

	response.Success(w, result.Message, result)
}
