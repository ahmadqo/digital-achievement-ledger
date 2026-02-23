package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/service"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
	"github.com/go-chi/chi/v5"
)

type StudentHandler struct {
	svc service.StudentService
}

func NewStudentHandler(svc service.StudentService) *StudentHandler {
	return &StudentHandler{svc: svc}
}

// GetAll retrieves all students with optional filters and pagination
// @Summary      Get all students
// @Description  Get a paginated list of students
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        search        query    string  false  "Search by NISN or name"
// @Param        class         query    string  false  "Filter by class"
// @Param        year_graduate query    int     false  "Filter by graduation year"
// @Param        page          query    int     false  "Page number (default 1)"
// @Param        per_page      query    int     false  "Items per page (default 10)"
// @Security     BearerAuth
// @Success      200  {object}  response.PaginatedResponse
// @Failure      500  {object}  response.Response
// @Router       /students [get]
func (h *StudentHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.StudentFilter{
		Search:  q.Get("search"),
		Class:   q.Get("class"),
		Page:    parseIntQuery(q.Get("page"), 1),
		PerPage: parseIntQuery(q.Get("per_page"), 10),
	}

	if y := q.Get("year_graduate"); y != "" {
		year, err := strconv.Atoi(y)
		if err == nil {
			filter.YearGraduate = &year
		}
	}

	students, pagination, err := h.svc.GetAll(r.Context(), filter)
	if err != nil {
		response.InternalError(w, "Gagal mengambil data siswa")
		return
	}

	response.Paginated(w, "Data siswa berhasil diambil", students, pagination)
}

// GetByID retrieves a student by ID
// @Summary      Get student by ID
// @Description  Get detailed information about a specific student
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Student ID"
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /students/{id} [get]
func (h *StudentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	student, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal mengambil data siswa")
		return
	}

	response.Success(w, "Data siswa berhasil diambil", student)
}

// Create adds a new student
// @Summary      Create a student
// @Description  Create a new student record
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        request  body      model.CreateStudentRequest  true  "Student creation request"
// @Security     BearerAuth
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /students [post]
func (h *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateStudentRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	errs := utils.ValidationErrors{}
	req.NISN = utils.SanitizeString(req.NISN)
	req.FullName = utils.SanitizeString(req.FullName)

	if req.NISN == "" {
		errs["nisn"] = "NISN wajib diisi"
	} else if len(req.NISN) != 10 {
		errs["nisn"] = "NISN harus 10 digit"
	}
	if req.FullName == "" {
		errs["full_name"] = "Nama lengkap wajib diisi"
	}
	if req.Gender != "" && req.Gender != "L" && req.Gender != "P" {
		errs["gender"] = "Gender harus L atau P"
	}

	if errs.HasErrors() {
		response.BadRequest(w, "Validasi gagal", errs)
		return
	}

	student, err := h.svc.Create(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrNISNAlreadyExist) {
			response.BadRequest(w, err.Error(), nil)
			return
		}
		response.InternalError(w, "Gagal membuat data siswa")
		return
	}

	response.Created(w, "Data siswa berhasil dibuat", student)
}

// Update modifies an existing student's data
// @Summary      Update a student
// @Description  Update details of an existing student
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id       path      string                      true  "Student ID"
// @Param        request  body      model.UpdateStudentRequest  true  "Student update request"
// @Security     BearerAuth
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /students/{id} [put]
func (h *StudentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateStudentRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	errs := utils.ValidationErrors{}
	req.FullName = utils.SanitizeString(req.FullName)
	if req.FullName == "" {
		errs["full_name"] = "Nama lengkap wajib diisi"
	}
	if errs.HasErrors() {
		response.BadRequest(w, "Validasi gagal", errs)
		return
	}

	student, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal mengupdate data siswa")
		return
	}

	response.Success(w, "Data siswa berhasil diupdate", student)
}

// Delete removes a student
// @Summary      Delete a student
// @Description  Delete a specific student and their associated photo/data
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Student ID"
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /students/{id} [delete]
func (h *StudentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Gagal menghapus data siswa")
		return
	}

	response.Success(w, "Data siswa berhasil dihapus", nil)
}

// UploadPhoto uploads or replaces a student's photo
// @Summary      Upload student photo
// @Description  Upload a JPEG or PNG photo for a student (max 5MB)
// @Tags         students
// @Accept       multipart/form-data
// @Produce      json
// @Param        id     path      string  true  "Student ID"
// @Param        photo  formData  file    true  "Photo file"
// @Security     BearerAuth
// @Success      200    {object}  response.Response
// @Failure      400    {object}  response.Response
// @Failure      404    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Router       /students/{id}/photo [post]
func (h *StudentHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024) // 5MB max
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
		response.BadRequest(w, "File terlalu besar atau format tidak valid", nil)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		response.BadRequest(w, "File foto tidak ditemukan dalam request", nil)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		response.BadRequest(w, "Format foto hanya JPG dan PNG", nil)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(w, "Gagal membaca file")
		return
	}

	student, err := h.svc.UploadPhoto(r.Context(), id, data, contentType)
	if err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.BadRequest(w, err.Error(), nil)
		return
	}

	response.Success(w, "Foto berhasil diupload", student)
}

func parseIntQuery(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	s = strings.TrimSpace(s)
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}
