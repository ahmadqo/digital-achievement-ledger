package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/ahmadqo/digital-achievement-ledger/internal/repository"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
)

var (
	ErrStudentNotFound  = errors.New("siswa tidak ditemukan")
	ErrNISNAlreadyExist = errors.New("NISN sudah terdaftar")
)

type StudentService interface {
	GetAll(ctx context.Context, filter model.StudentFilter) ([]*model.Student, *response.Pagination, error)
	GetByID(ctx context.Context, id string) (*model.Student, error)
	Create(ctx context.Context, req model.CreateStudentRequest) (*model.Student, error)
	Update(ctx context.Context, id string, req model.UpdateStudentRequest) (*model.Student, error)
	Delete(ctx context.Context, id string) error
	UploadPhoto(ctx context.Context, id string, data []byte, contentType string) (*model.Student, error)
}

type studentService struct {
	repo    repository.StudentRepository
	storage *utils.StorageService
}

func NewStudentService(repo repository.StudentRepository, storage *utils.StorageService) StudentService {
	return &studentService{repo: repo, storage: storage}
}

func (s *studentService) GetAll(ctx context.Context, filter model.StudentFilter) ([]*model.Student, *response.Pagination, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 10
	}

	students, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	totalPages := int(total) / filter.PerPage
	if int(total)%filter.PerPage > 0 {
		totalPages++
	}

	pagination := &response.Pagination{
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalItems: total,
		TotalPages: totalPages,
	}

	return students, pagination, nil
}

func (s *studentService) GetByID(ctx context.Context, id string) (*model.Student, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("ID tidak valid")
	}

	student, err := s.repo.FindByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentNotFound
	}

	return student, nil
}

func (s *studentService) Create(ctx context.Context, req model.CreateStudentRequest) (*model.Student, error) {
	// Cek NISN duplikat
	existing, err := s.repo.FindByNISN(ctx, req.NISN)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrNISNAlreadyExist
	}

	student := &model.Student{
		ID:           uuid.New(),
		NISN:         req.NISN,
		FullName:     req.FullName,
		BirthPlace:   req.BirthPlace,
		Gender:       req.Gender,
		Class:        req.Class,
		YearEntry:    req.YearEntry,
		YearGraduate: req.YearGraduate,
	}

	// Parse birth date
	if req.BirthDate != "" {
		t, err := time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			return nil, errors.New("format tanggal lahir tidak valid, gunakan YYYY-MM-DD")
		}
		student.BirthDate = &t
	}

	if err := s.repo.Create(ctx, student); err != nil {
		return nil, err
	}

	return student, nil
}

func (s *studentService) Update(ctx context.Context, id string, req model.UpdateStudentRequest) (*model.Student, error) {
	student, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	student.FullName = req.FullName
	student.BirthPlace = req.BirthPlace
	student.Gender = req.Gender
	student.Class = req.Class
	student.YearEntry = req.YearEntry
	student.YearGraduate = req.YearGraduate

	if req.BirthDate != "" {
		t, err := time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			return nil, errors.New("format tanggal lahir tidak valid, gunakan YYYY-MM-DD")
		}
		student.BirthDate = &t
	}

	if err := s.repo.Update(ctx, student); err != nil {
		return nil, err
	}

	return student, nil
}

func (s *studentService) Delete(ctx context.Context, id string) error {
	_, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	uid, _ := uuid.Parse(id)
	return s.repo.Delete(ctx, uid)
}

func (s *studentService) UploadPhoto(ctx context.Context, id string, data []byte, contentType string) (*model.Student, error) {
	student, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Hapus foto lama jika ada
	if student.PhotoURL != nil {
		s.storage.DeleteFile(ctx, *student.PhotoURL)
	}

	result, err := s.storage.UploadFile(ctx, "students/photos", data, contentType)
	if err != nil {
		return nil, err
	}

	uid, _ := uuid.Parse(id)
	if err := s.repo.UpdatePhoto(ctx, uid, result.FileURL); err != nil {
		return nil, err
	}

	student.PhotoURL = &result.FileURL
	return student, nil
}