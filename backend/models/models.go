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
	ADUserID     sql.NullString `json:"ad_user_id"` // ID пользователя в AD
	IsADUser     bool           `json:"is_ad_user"` // Флаг, что это пользователь из AD
}

type ADConfig struct {
	ID              int       `json:"id"`
	Enabled         bool      `json:"enabled"`
	DirectoryType   string    `json:"directory_type"`    // "active_directory" или "freeipa"
	ServerURL       string    `json:"server_url"`        // LDAP server URL
	BaseDN          string    `json:"base_dn"`           // Base DN
	BindDN          string    `json:"bind_dn"`           // Bind DN for service account
	BindPassword    string    `json:"bind_password"`     // Bind password
	UserSearchBase  string    `json:"user_search_base"`  // OU для поиска пользователей
	UserNameAttr    string    `json:"user_name_attr"`    // Атрибут для имени (sAMAccountName или uid)
	DepartmentAttr  string    `json:"department_attr"`   // Атрибут для отдела (department)
	EmailAttr       string    `json:"email_attr"`        // Атрибут для email
	GroupSearchBase string    `json:"group_search_base"` // OU для поиска групп
	SyncInterval    int       `json:"sync_interval"`     // Интервал синхронизации в минутах
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
