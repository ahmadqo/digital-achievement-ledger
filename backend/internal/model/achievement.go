package model

import (
	"time"

	"github.com/google/uuid"
)

type Achievement struct {
	ID              uuid.UUID  `db:"id"               json:"id"`
	StudentID       uuid.UUID  `db:"student_id"       json:"student_id"`
	CompetitionName string     `db:"competition_name" json:"competition_name"`
	Organizer       string     `db:"organizer"        json:"organizer"`
	CategoryID      *int       `db:"category_id"      json:"category_id"`
	Rank            string     `db:"rank"             json:"rank"`
	LevelID         *int       `db:"level_id"         json:"level_id"`
	Year            int        `db:"year"             json:"year"`
	Description     string     `db:"description"      json:"description"`
	CreatedBy       *uuid.UUID `db:"created_by"       json:"created_by"`
	CreatedAt       time.Time  `db:"created_at"       json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"       json:"updated_at"`

	// Join fields
	CategoryName *string `db:"category_name" json:"category_name,omitempty"`
	LevelName    *string `db:"level_name"    json:"level_name,omitempty"`
	StudentName  *string `db:"student_name"  json:"student_name,omitempty"`
	StudentNISN  *string `db:"student_nisn"  json:"student_nisn,omitempty"`
}

type AchievementWithAttachments struct {
	Achievement
	Attachments []AchievementAttachment `json:"attachments"`
}

type AchievementAttachment struct {
	ID            uuid.UUID `db:"id"             json:"id"`
	AchievementID uuid.UUID `db:"achievement_id" json:"achievement_id"`
	FileURL       string    `db:"file_url"       json:"file_url"`
	FileName      string    `db:"file_name"      json:"file_name"`
	FileType      string    `db:"file_type"      json:"file_type"`
	Label         string    `db:"label"          json:"label"`
	UploadedAt    time.Time `db:"uploaded_at"    json:"uploaded_at"`
}

type CreateAchievementRequest struct {
	StudentID       string `json:"student_id"`
	CompetitionName string `json:"competition_name"`
	Organizer       string `json:"organizer"`
	CategoryID      *int   `json:"category_id"`
	Rank            string `json:"rank"`
	LevelID         *int   `json:"level_id"`
	Year            int    `json:"year"`
	Description     string `json:"description"`
}

type UpdateAchievementRequest struct {
	CompetitionName string `json:"competition_name"`
	Organizer       string `json:"organizer"`
	CategoryID      *int   `json:"category_id"`
	Rank            string `json:"rank"`
	LevelID         *int   `json:"level_id"`
	Year            int    `json:"year"`
	Description     string `json:"description"`
}

type AchievementFilter struct {
	StudentID  string
	CategoryID *int
	LevelID    *int
	Year       *int
	Search     string
	Page       int
	PerPage    int
}

type AchievementCategory struct {
	ID   int    `db:"id"   json:"id"`
	Name string `db:"name" json:"name"`
	Type string `db:"type" json:"type"`
}

type CompetitionLevel struct {
	ID        int    `db:"id"         json:"id"`
	Name      string `db:"name"       json:"name"`
	OrderRank int    `db:"order_rank" json:"order_rank"`
}