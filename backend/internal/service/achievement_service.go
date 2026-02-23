package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/ahmadqo/digital-achievement-ledger/internal/repository"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
)

var ErrAchievementNotFound = errors.New("prestasi tidak ditemukan")

type AchievementService interface {
	GetAll(ctx context.Context, filter model.AchievementFilter) ([]*model.Achievement, *response.Pagination, error)
	GetByID(ctx context.Context, id string) (*model.AchievementWithAttachments, error)
	Create(ctx context.Context, req model.CreateAchievementRequest, createdBy string) (*model.Achievement, error)
	Update(ctx context.Context, id string, req model.UpdateAchievementRequest) (*model.Achievement, error)
	Delete(ctx context.Context, id string) error
	UploadAttachment(ctx context.Context, achievementID string, data []byte, contentType, label string) (*model.AchievementAttachment, error)
	DeleteAttachment(ctx context.Context, attachmentID string) error
	GetCategories(ctx context.Context) ([]*model.AchievementCategory, error)
	GetLevels(ctx context.Context) ([]*model.CompetitionLevel, error)
}

type achievementService struct {
	repo        repository.AchievementRepository
	studentRepo repository.StudentRepository
	storage     *utils.StorageService
}

func NewAchievementService(
	repo repository.AchievementRepository,
	studentRepo repository.StudentRepository,
	storage *utils.StorageService,
) AchievementService {
	return &achievementService{repo: repo, studentRepo: studentRepo, storage: storage}
}

func (s *achievementService) GetAll(ctx context.Context, filter model.AchievementFilter) ([]*model.Achievement, *response.Pagination, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 10
	}

	achievements, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	totalPages := int(total) / filter.PerPage
	if int(total)%filter.PerPage > 0 {
		totalPages++
	}

	return achievements, &response.Pagination{
		Page: filter.Page, PerPage: filter.PerPage,
		TotalItems: total, TotalPages: totalPages,
	}, nil
}

func (s *achievementService) GetByID(ctx context.Context, id string) (*model.AchievementWithAttachments, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("ID tidak valid")
	}

	achievement, err := s.repo.FindByIDWithAttachments(ctx, uid)
	if err != nil {
		return nil, err
	}
	if achievement == nil {
		return nil, ErrAchievementNotFound
	}

	return achievement, nil
}

func (s *achievementService) Create(ctx context.Context, req model.CreateAchievementRequest, createdBy string) (*model.Achievement, error) {
	studentUID, err := uuid.Parse(req.StudentID)
	if err != nil {
		return nil, errors.New("student_id tidak valid")
	}

	// Pastikan student ada
	student, err := s.studentRepo.FindByID(ctx, studentUID)
	if err != nil || student == nil {
		return nil, ErrStudentNotFound
	}

	createdByUID, _ := uuid.Parse(createdBy)

	achievement := &model.Achievement{
		ID:              uuid.New(),
		StudentID:       studentUID,
		CompetitionName: req.CompetitionName,
		Organizer:       req.Organizer,
		CategoryID:      req.CategoryID,
		Rank:            req.Rank,
		LevelID:         req.LevelID,
		Year:            req.Year,
		Description:     req.Description,
		CreatedBy:       &createdByUID,
	}

	if err := s.repo.Create(ctx, achievement); err != nil {
		return nil, err
	}

	return achievement, nil
}

func (s *achievementService) Update(ctx context.Context, id string, req model.UpdateAchievementRequest) (*model.Achievement, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("ID tidak valid")
	}

	achievement, err := s.repo.FindByID(ctx, uid)
	if err != nil || achievement == nil {
		return nil, ErrAchievementNotFound
	}

	achievement.CompetitionName = req.CompetitionName
	achievement.Organizer = req.Organizer
	achievement.CategoryID = req.CategoryID
	achievement.Rank = req.Rank
	achievement.LevelID = req.LevelID
	achievement.Year = req.Year
	achievement.Description = req.Description

	if err := s.repo.Update(ctx, achievement); err != nil {
		return nil, err
	}

	return achievement, nil
}

func (s *achievementService) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.New("ID tidak valid")
	}

	achievement, err := s.repo.FindByIDWithAttachments(ctx, uid)
	if err != nil || achievement == nil {
		return ErrAchievementNotFound
	}

	// Hapus semua file attachment dari MinIO
	for _, att := range achievement.Attachments {
		s.storage.DeleteFile(ctx, att.FileURL)
	}

	return s.repo.Delete(ctx, uid)
}

func (s *achievementService) UploadAttachment(ctx context.Context, achievementID string, data []byte, contentType, label string) (*model.AchievementAttachment, error) {
	uid, err := uuid.Parse(achievementID)
	if err != nil {
		return nil, errors.New("achievement_id tidak valid")
	}

	achievement, err := s.repo.FindByID(ctx, uid)
	if err != nil || achievement == nil {
		return nil, ErrAchievementNotFound
	}

	result, err := s.storage.UploadFile(ctx, "achievements/attachments", data, contentType)
	if err != nil {
		return nil, err
	}

	if label == "" {
		label = "Bukti Prestasi"
	}

	att := &model.AchievementAttachment{
		ID:            uuid.New(),
		AchievementID: uid,
		FileURL:       result.FileURL,
		FileName:      result.FileName,
		FileType:      contentType,
		Label:         label,
	}

	if err := s.repo.AddAttachment(ctx, att); err != nil {
		s.storage.DeleteFile(ctx, result.FileURL) // rollback file jika DB gagal
		return nil, err
	}

	return att, nil
}

func (s *achievementService) DeleteAttachment(ctx context.Context, attachmentID string) error {
	uid, err := uuid.Parse(attachmentID)
	if err != nil {
		return errors.New("ID tidak valid")
	}

	att, err := s.repo.DeleteAttachment(ctx, uid)
	if err != nil {
		return err
	}
	if att == nil {
		return errors.New("attachment tidak ditemukan")
	}

	// Hapus file dari MinIO
	s.storage.DeleteFile(ctx, att.FileURL)
	return nil
}

func (s *achievementService) GetCategories(ctx context.Context) ([]*model.AchievementCategory, error) {
	return s.repo.FindAllCategories(ctx)
}

func (s *achievementService) GetLevels(ctx context.Context) ([]*model.CompetitionLevel, error) {
	return s.repo.FindAllLevels(ctx)
}