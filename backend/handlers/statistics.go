package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type StatisticsHandler struct {
	db *sql.DB
}

type EmployeeStatistics struct {
	ID                int     `json:"id"`
	Username          string  `json:"username"`
	Role              string  `json:"role"`
	Department        string  `json:"department"`
	HasTasks          bool    `json:"has_tasks"`
	TotalLoadPerMonth int     `json:"total_load_per_month"`
	TotalHoursPerWeek float64 `json:"total_hours_per_week"`
}

func NewStatisticsHandler(db *sql.DB) *StatisticsHandler {
	return &StatisticsHandler{db: db}
}

// GetDepartmentStatistics возвращает статистику всех сотрудников отдела менеджера
func (h *StatisticsHandler) GetDepartmentStatistics(c *gin.Context) {
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	// Только менеджеры и админы могут видеть статистику отдела
	if userRole != "manager" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Получаем параметры дат
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" && endDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
			return
		}
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
			return
		}
		// Устанавливаем конец дня для end_date
		endDate = endDate.Add(time.Hour * 23).Add(time.Minute * 59).Add(time.Second * 59)
	}

	var rows *sql.Rows

	if userRole == "admin" {
		// Админ видит всех пользователей
		rows, err = h.db.Query(`
			SELECT DISTINCT u.id, u.username, u.role, u.department, u.first_name, u.last_name, u.patronymic
			FROM users u
			ORDER BY u.department, u.last_name, u.first_name
		`)
	} else {
		// Менеджер видит только пользователей своего отдела
		rows, err = h.db.Query(`
			SELECT DISTINCT u.id, u.username, u.role, u.department, u.first_name, u.last_name, u.patronymic
			FROM users u
			WHERE u.department = ?
			ORDER BY u.last_name, u.first_name
		`, userDepartment)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	stats := []EmployeeStatistics{}

	for rows.Next() {
		var emp EmployeeStatistics
		var dept sql.NullString
		var firstName, lastName, patronymic sql.NullString

		err := rows.Scan(&emp.ID, &emp.Username, &emp.Role, &dept, &firstName, &lastName, &patronymic)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if dept.Valid {
			emp.Department = dept.String
		}

		// Формируем ФИО и сохраняем в Username для отправки на фронт
		fullName := ""
		if lastName.Valid && lastName.String != "" && firstName.Valid && firstName.String != "" {
			fullName = lastName.String + " " + firstName.String
			if patronymic.Valid && patronymic.String != "" {
				fullName += " " + patronymic.String
			}
		}
		if fullName != "" {
			emp.Username = fullName
		}

		// Получаем статистику по задачам для этого пользователя
		var taskCount int
		var totalLoadPerMonth sql.NullInt64
		var totalHoursPerWeek sql.NullFloat64

		query := `
			SELECT 
				COUNT(*),
				COALESCE(SUM(load_per_month), 0),
				COALESCE(SUM(hours_per_week), 0)
			FROM tasks
			WHERE user_id = ?`

		args := []interface{}{emp.ID}

		if !startDate.IsZero() && !endDate.IsZero() {
			query += ` AND created_at >= ? AND created_at <= ?`
			args = append(args, startDate, endDate)
		}

		err = h.db.QueryRow(query, args...).Scan(&taskCount, &totalLoadPerMonth, &totalHoursPerWeek)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		emp.HasTasks = taskCount > 0
		if totalLoadPerMonth.Valid {
			emp.TotalLoadPerMonth = int(totalLoadPerMonth.Int64)
		}
		if totalHoursPerWeek.Valid {
			emp.TotalHoursPerWeek = totalHoursPerWeek.Float64
		}

		stats = append(stats, emp)
	}

	if stats == nil {
		stats = []EmployeeStatistics{}
	}

	c.JSON(http.StatusOK, stats)
}

// ClearDepartmentTasks удаляет все задачи сотрудников отдела менеджера
func (h *StatisticsHandler) ClearDepartmentTasks(c *gin.Context) {
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	// Только менеджеры могут очищать задачи своего отдела
	if userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only managers can clear department tasks"})
		return
	}

	// Удаляем все задачи пользователей из отдела менеджера
	result, err := h.db.Exec(`
		DELETE FROM tasks
		WHERE user_id IN (
			SELECT id FROM users
			WHERE department = ?
		)
	`, userDepartment)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Department tasks cleared successfully",
		"deleted": rowsAffected,
	})
}

// ExportDepartmentStatistics экспортирует статистику отдела в Excel
func (h *StatisticsHandler) ExportDepartmentStatistics(c *gin.Context) {
	userRole := c.GetString("userRole")

	// Только менеджеры и админы могут экспортировать
	if userRole != "manager" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Для совместимости с существующей функцией ExportDepartmentTasks
	c.Redirect(http.StatusMovedPermanently, "/api/reports/department-tasks")
}

// GetWeeklyHours возвращает количество часов за последнюю неделю для конкретного пользователя
func (h *StatisticsHandler) GetWeeklyHours(c *gin.Context) {
	currentUserID := c.GetInt("userID")

	userID := currentUserID

	// Менеджеры и админы могут видеть часы других пользователей своего отдела по параметру
	userRole := c.GetString("userRole")
	if userIDStr, exists := c.GetQuery("user_id"); exists && (userRole == "manager" || userRole == "admin") {
		userDepartment := c.GetString("userDepartment")
		var checkDept sql.NullString
		err := h.db.QueryRow("SELECT department FROM users WHERE id = ?", userIDStr).Scan(&checkDept)
		if err == nil && checkDept.Valid && checkDept.String == userDepartment {
			// Парсим ID из параметра, если пользователь в том же отделе
			_, _ = fmt.Sscanf(userIDStr, "%d", &userID)
		}
	}

	// Получаем часы за последнюю неделю
	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	var totalHours sql.NullFloat64
	err := h.db.QueryRow(`
		SELECT COALESCE(SUM(hours_per_week), 0)
		FROM tasks
		WHERE user_id = ? AND updated_at >= ?
	`, userID, oneWeekAgo).Scan(&totalHours)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	hours := 0.0
	if totalHours.Valid {
		hours = totalHours.Float64
	}

	c.JSON(http.StatusOK, gin.H{"hours_per_week": hours})
}
