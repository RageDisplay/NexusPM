package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int            `json:"id"`
	Username     string         `json:"username"`
	PasswordHash string         `json:"-"`
	Role         string         `json:"role"`
	Department   sql.NullString `json:"department"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserProject struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	ProjectID int       `json:"project_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Task struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Progress     int       `json:"progress"`
	HoursPerWeek float64   `json:"hours_per_week"`
	LoadPerMonth int       `json:"load_per_month"`
	UserID       int       `json:"user_id"`
	ProjectID    int       `json:"project_id"`
	Username     string    `json:"username,omitempty"`
	Department   string    `json:"department,omitempty"`
	ProjectName  string    `json:"project_name,omitempty"`
	WeeklyInfo   string    `json:"weekly_info"`
	Planning     string    `json:"planning"`
	HelpNeeded   string    `json:"help_needed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
