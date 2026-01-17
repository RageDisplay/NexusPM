package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	db *sql.DB
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

type TaskStatistics struct {
	TotalTasks        int     `json:"total_tasks"`
	ActiveTasks       int     `json:"active_tasks"`
	CompletedTasks    int     `json:"completed_tasks"`
	AverageProgress   float64 `json:"average_progress"`
	TotalHoursPerWeek float64 `json:"total_hours_per_week"`
	TotalLoadPerMonth int     `json:"total_load_per_month"`
}

type LoadByWeek struct {
	Week      int     `json:"week"`
	LoadHours float64 `json:"load_hours"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
}

type ProgressOverTime struct {
	Date     string  `json:"date"`
	Progress float64 `json:"progress"`
}

type ActivityLog struct {
	ID          int       `json:"id"`
	ActionType  string    `json:"action_type"`
	Description string    `json:"description"`
	TaskID      *int      `json:"task_id"`
	ProjectID   *int      `json:"project_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type Notification struct {
	ID               int       `json:"id"`
	NotificationType string    `json:"notification_type"`
	Title            string    `json:"title"`
	Message          string    `json:"message"`
	RelatedID        *int      `json:"related_id"`
	IsRead           bool      `json:"is_read"`
	CreatedAt        time.Time `json:"created_at"`
}

// GetTaskStatistics возвращает статистику задач пользователя
func (h *DashboardHandler) GetTaskStatistics(c *gin.Context) {
	userID := c.GetInt("userID")

	var stats TaskStatistics

	err := h.db.QueryRow(`
		SELECT 
			COUNT(*),
			SUM(CASE WHEN progress < 100 THEN 1 ELSE 0 END),
			SUM(CASE WHEN progress = 100 THEN 1 ELSE 0 END),
			COALESCE(AVG(progress), 0),
			COALESCE(SUM(hours_per_week), 0),
			COALESCE(SUM(load_per_month), 0)
		FROM tasks
		WHERE user_id = ?
	`, userID).Scan(
		&stats.TotalTasks,
		&stats.ActiveTasks,
		&stats.CompletedTasks,
		&stats.AverageProgress,
		&stats.TotalHoursPerWeek,
		&stats.TotalLoadPerMonth,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetWeeklyLoad возвращает загруженность по неделям текущего месяца
func (h *DashboardHandler) GetWeeklyLoad(c *gin.Context) {
	userID := c.GetInt("userID")

	rows, err := h.db.Query(`
		SELECT 
			CAST((CAST(strftime('%d', created_at) AS INTEGER) - 1) / 7 + 1 AS INTEGER) as week,
			COALESCE(SUM(hours_per_week), 0) as load_hours,
			MIN(DATE(created_at)) as start_date,
			MAX(DATE(created_at)) as end_date
		FROM tasks
		WHERE user_id = ? 
		AND strftime('%Y-%m', created_at) = strftime('%Y-%m', 'now')
		GROUP BY week
		ORDER BY week
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var loads []LoadByWeek
	for rows.Next() {
		var load LoadByWeek
		if err := rows.Scan(&load.Week, &load.LoadHours, &load.StartDate, &load.EndDate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		loads = append(loads, load)
	}

	if loads == nil {
		loads = []LoadByWeek{}
	}

	c.JSON(http.StatusOK, loads)
}

// GetProgressDynamics возвращает динамику прогресса по датам
func (h *DashboardHandler) GetProgressDynamics(c *gin.Context) {
	userID := c.GetInt("userID")

	rows, err := h.db.Query(`
		SELECT 
			DATE(updated_at) as date,
			ROUND(AVG(progress), 2) as progress
		FROM tasks
		WHERE user_id = ?
		AND updated_at >= datetime('now', '-30 days')
		GROUP BY DATE(updated_at)
		ORDER BY date
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var progressData []ProgressOverTime
	for rows.Next() {
		var p ProgressOverTime
		if err := rows.Scan(&p.Date, &p.Progress); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		progressData = append(progressData, p)
	}

	if progressData == nil {
		progressData = []ProgressOverTime{}
	}

	c.JSON(http.StatusOK, progressData)
}

// GetActivityLog возвращает историю активностей пользователя
func (h *DashboardHandler) GetActivityLog(c *gin.Context) {
	userID := c.GetInt("userID")
	limit := 50

	rows, err := h.db.Query(`
		SELECT id, action_type, description, task_id, project_id, created_at
		FROM activity_log
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var activities []ActivityLog
	for rows.Next() {
		var activity ActivityLog
		if err := rows.Scan(&activity.ID, &activity.ActionType, &activity.Description, &activity.TaskID, &activity.ProjectID, &activity.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		activities = append(activities, activity)
	}

	if activities == nil {
		activities = []ActivityLog{}
	}

	c.JSON(http.StatusOK, activities)
}

// GetNotifications возвращает уведомления пользователя
func (h *DashboardHandler) GetNotifications(c *gin.Context) {
	userID := c.GetInt("userID")

	rows, err := h.db.Query(`
		SELECT id, notification_type, title, message, related_id, is_read, created_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 20
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notif Notification
		if err := rows.Scan(&notif.ID, &notif.NotificationType, &notif.Title, &notif.Message, &notif.RelatedID, &notif.IsRead, &notif.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		notifications = append(notifications, notif)
	}

	if notifications == nil {
		notifications = []Notification{}
	}

	c.JSON(http.StatusOK, notifications)
}

// MarkNotificationAsRead отмечает уведомление как прочитанное
func (h *DashboardHandler) MarkNotificationAsRead(c *gin.Context) {
	userID := c.GetInt("userID")
	notificationID := c.Param("id")

	_, err := h.db.Exec(`
		UPDATE notifications
		SET is_read = 1
		WHERE id = ? AND user_id = ?
	`, notificationID, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// LogActivity записывает действие в лог активности
func LogActivity(db *sql.DB, userID int, actionType, description string, taskID, projectID *int) error {
	_, err := db.Exec(`
		INSERT INTO activity_log (user_id, action_type, description, task_id, project_id)
		VALUES (?, ?, ?, ?, ?)
	`, userID, actionType, description, taskID, projectID)
	return err
}

// CreateNotification создает уведомление для пользователя
func CreateNotification(db *sql.DB, userID int, notificationType, title, message string, relatedID *int) error {
	_, err := db.Exec(`
		INSERT INTO notifications (user_id, notification_type, title, message, related_id)
		VALUES (?, ?, ?, ?, ?)
	`, userID, notificationType, title, message, relatedID)
	return err
}
