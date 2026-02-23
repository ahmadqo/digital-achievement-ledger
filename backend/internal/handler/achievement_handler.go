package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/ahmadqo/digital-achievement-ledger/internal/middleware"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/service"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
	"github.com/go-chi/chi/v5"
)

type AchievementHandler struct {
	svc service.AchievementService
}

func NewAchievementHandler(svc service.AchievementService) *AchievementHandler {
	return &AchievementHandler{svc: svc}
}

// GetAll retrieves all achievements with optional filters
// @Summary      Get all achievements
// @Description  Get a paginated list of achievements
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Param        student_id   query    string  false  "Filter by student ID"
// @Param        search       query    string  false  "Search by competition name"
// @Param        category_id  query    int     false  "Filter by category ID"
// @Param        level_id     query    int     false  "Filter by level ID"
// @Param        year         query    int     false  "Filter by year"
// @Param        page         query    int     false  "Page number"
// @Param        per_page     query    int     false  "Items per page"
// @Security     BearerAuth
// @Success      200  {object}  response.PaginatedResponse
// @Failure      500  {object}  response.Response
// @Router       /achievements [get]
func (h *AchievementHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.AchievementFilter{
		StudentID: q.Get("student_id"),
		Search:    q.Get("search"),
		Page:      parseIntQuery(q.Get("page"), 1),
		PerPage:   parseIntQuery(q.Get("per_page"), 10),
	}

	if c := q.Get("category_id"); c != "" {
		if v, err := strconv.Atoi(c); err == nil {
			filter.CategoryID = &v
		}
	}
	if l := q.Get("level_id"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			filter.LevelID = &v
		}
	}
	if y := q.Get("year"); y != "" {
		if v, err := strconv.Atoi(y); err == nil {
			filter.Year = &v
		}
	}

	achievements, pagination, err := h.svc.GetAll(r.Context(), filter)
	if err != nil {
		response.InternalError(w, "Gagal mengambil data prestasi")
		return
	}

	response.Paginated(w, "Data prestasi berhasil diambil", achievements, pagination)
}

// GetByID retrieves a specific achievement
// @Summary      Get achievement by ID
// @Description  Get detailed info about a specific achievement
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Achievement ID"
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /achievements/{id} [get]
func (h *AchievementHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	achievement, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAchievementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal mengambil data prestasi")
		return
	}

	response.Success(w, "Data prestasi berhasil diambil", achievement)
}

// Create adds a new achievement
// @Summary      Create an achievement
// @Description  Create a new achievement record for a student
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Param        request  body      model.CreateAchievementRequest  true  "Achievement creation request"
// @Security     BearerAuth
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /achievements [post]
func (h *AchievementHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateAchievementRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	errs := utils.ValidationErrors{}
	req.CompetitionName = utils.SanitizeString(req.CompetitionName)
	req.Organizer = utils.SanitizeString(req.Organizer)

	if req.StudentID == "" {
		errs["student_id"] = "Student ID wajib diisi"
	}
	if req.CompetitionName == "" {
		errs["competition_name"] = "Nama lomba wajib diisi"
	}
	if req.Organizer == "" {
		errs["organizer"] = "Penyelenggara wajib diisi"
	}
	if req.Rank == "" {
		errs["rank"] = "Juara/peringkat wajib diisi"
	}
	if req.Year == 0 {
		errs["year"] = "Tahun wajib diisi"
	}

	if errs.HasErrors() {
		response.BadRequest(w, "Validasi gagal", errs)
		return
	}

	createdBy := middleware.GetUserIDFromContext(r.Context())
	achievement, err := h.svc.Create(r.Context(), req, createdBy)
	if err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal membuat data prestasi")
		return
	}

	response.Created(w, "Data prestasi berhasil dibuat", achievement)
}

// Update modifies an existing achievement
// @Summary      Update an achievement
// @Description  Update details of an existing achievement
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Param        id       path      string                          true  "Achievement ID"
// @Param        request  body      model.UpdateAchievementRequest  true  "Achievement update request"
// @Security     BearerAuth
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /achievements/{id} [put]
func (h *AchievementHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateAchievementRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	errs := utils.ValidationErrors{}
	if req.CompetitionName == "" {
		errs["competition_name"] = "Nama lomba wajib diisi"
	}
	if req.Organizer == "" {
		errs["organizer"] = "Penyelenggara wajib diisi"
	}
	if errs.HasErrors() {
		response.BadRequest(w, "Validasi gagal", errs)
		return
	}

	achievement, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrAchievementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal mengupdate data prestasi")
		return
	}

	response.Success(w, "Data prestasi berhasil diupdate", achievement)
}

// Delete removes an achievement
// @Summary      Delete an achievement
// @Description  Delete a specific achievement and all its attachments
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Achievement ID"
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /achievements/{id} [delete]
func (h *AchievementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrAchievementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal menghapus data prestasi")
		return
	}

	response.Success(w, "Data prestasi berhasil dihapus", nil)
}

// UploadAttachment adds a file attachment to an achievement
// @Summary      Upload achievement attachment
// @Description  Upload an attachment (image/pdf) for an achievement
// @Tags         achievements
// @Accept       multipart/form-data
// @Produce      json
// @Param        id     path      string  true   "Achievement ID"
// @Param        file   formData  file    true   "Attachment file (PDF, JPG, PNG)"
// @Param        label  formData  string  false  "Optional label/description for the attachment"
// @Security     BearerAuth
// @Success      201    {object}  response.Response
// @Failure      400    {object}  response.Response
// @Failure      404    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Router       /achievements/{id}/attachments [post]
func (h *AchievementHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB max
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		response.BadRequest(w, "File terlalu besar (max 10MB) atau format tidak valid", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "File tidak ditemukan dalam request", nil)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if _, ok := utils.AllowedAttachmentTypes[contentType]; !ok {
		response.BadRequest(w, "Format file tidak didukung. Gunakan JPG, PNG, atau PDF", nil)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(w, "Gagal membaca file")
		return
	}

	label := r.FormValue("label")

	att, err := h.svc.UploadAttachment(r.Context(), id, data, contentType, label)
	if err != nil {
		if errors.Is(err, service.ErrAchievementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.BadRequest(w, err.Error(), nil)
		return
	}

	response.Created(w, "Attachment berhasil diupload", att)
}

// DeleteAttachment removes a specific file attachment
// @Summary      Delete achievement attachment
// @Description  Remove an attachment from an achievement
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Param        attachmentId  path      string  true  "Attachment ID"
// @Security     BearerAuth
// @Success      200           {object}  response.Response
// @Failure      400           {object}  response.Response
// @Router       /achievements/attachments/{attachmentId} [delete]
func (h *AchievementHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	attachmentID := chi.URLParam(r, "attachmentId")

	if err := h.svc.DeleteAttachment(r.Context(), attachmentID); err != nil {
		response.BadRequest(w, err.Error(), nil)
		return
	}

	response.Success(w, "Attachment berhasil dihapus", nil)
}

// GetCategories retrieves the list of category enums
// @Summary      Get achievement categories
// @Description  Get a list of available achievement categories (e.g., Akademik, Olahraga)
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /achievements/categories [get]
func (h *AchievementHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.GetCategories(r.Context())
	if err != nil {
		response.InternalError(w, "Gagal mengambil data kategori")
		return
	}
	response.Success(w, "Data kategori berhasil diambil", categories)
}

// GetLevels retrieves the list of level enums
// @Summary      Get achievement levels
// @Description  Get a list of available achievement levels (e.g., Nasional, Provinsi)
// @Tags         achievements
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /achievements/levels [get]
func (h *AchievementHandler) GetLevels(w http.ResponseWriter, r *http.Request) {
	levels, err := h.svc.GetLevels(r.Context())
	if err != nil {
		response.InternalError(w, "Gagal mengambil data tingkat lomba")
		return
	}
	response.Success(w, "Data tingkat lomba berhasil diambil", levels)
}
