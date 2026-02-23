package model

import (
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID           uuid.UUID  `db:"id"            json:"id"`
	NISN         string     `db:"nisn"          json:"nisn"`
	FullName     string     `db:"full_name"     json:"full_name"`
	BirthPlace   string     `db:"birth_place"   json:"birth_place"`
	BirthDate    *time.Time `db:"birth_date"    json:"birth_date"`
	Gender       string     `db:"gender"        json:"gender"`
	Class        string     `db:"class"         json:"class"`
	YearEntry    *int       `db:"year_entry"    json:"year_entry"`
	YearGraduate *int       `db:"year_graduate" json:"year_graduate"`
	PhotoURL     *string    `db:"photo_url"     json:"photo_url"`
	CreatedAt    time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"    json:"updated_at"`
}

type CreateStudentRequest struct {
	NISN         string  `json:"nisn"`
	FullName     string  `json:"full_name"`
	BirthPlace   string  `json:"birth_place"`
	BirthDate    string  `json:"birth_date"` // format: YYYY-MM-DD
	Gender       string  `json:"gender"`     // L | P
	Class        string  `json:"class"`
	YearEntry    *int    `json:"year_entry"`
	YearGraduate *int    `json:"year_graduate"`
}

type UpdateStudentRequest struct {
	FullName     string  `json:"full_name"`
	BirthPlace   string  `json:"birth_place"`
	BirthDate    string  `json:"birth_date"`
	Gender       string  `json:"gender"`
	Class        string  `json:"class"`
	YearEntry    *int    `json:"year_entry"`
	YearGraduate *int    `json:"year_graduate"`
}

type StudentFilter struct {
	Search       string
	Class        string
	YearGraduate *int
	Page         int
	PerPage      int
}