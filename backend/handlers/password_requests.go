package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"
	"time"

	"task-management-backend/database"

	"github.com/gin-gonic/gin"
)

type PasswordRequestHandler struct {
	db *sql.DB
}

func NewPasswordRequestHandler(db *sql.DB) *PasswordRequestHandler {
	return &PasswordRequestHandler{db: db}
}

// Создать заявку на сброс пароля
func (h *PasswordRequestHandler) CreateRequest(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Попытка найти пользователя
	var userID sql.NullInt64
	var foundUsername sql.NullString
	err := h.db.QueryRow("SELECT id, username FROM users WHERE username = ?", req.Username).Scan(&userID, &foundUsername)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Отправляем заявку, даже если пользователь не найден, чтобы не раскрывать информацию
	_, err = h.db.Exec("INSERT INTO password_reset_requests (user_id, username, message, status, created_at) VALUES (?, ?, ?, 'open', ?)",
		userID, req.Username, req.Message, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Для безопасности не сообщаем, найден пользователь или нет
	c.JSON(http.StatusOK, gin.H{"message": "Заявка отправлена. Администратор рассмотрит запрос."})
}

// Админ: список открытых заявок
func (h *PasswordRequestHandler) ListRequests(c *gin.Context) {
	rows, err := h.db.Query("SELECT id, user_id, username, message, status, created_at, processed_by, processed_at FROM password_reset_requests WHERE status = 'open' ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id int
		var userID sql.NullInt64
		var username sql.NullString
		var message sql.NullString
		var status string
		var createdAt string
		var processedBy sql.NullInt64
		var processedAt sql.NullString

		if err := rows.Scan(&id, &userID, &username, &message, &status, &createdAt, &processedBy, &processedAt); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id":         id,
			"user_id":    userID.Int64,
			"username":   username.String,
			"message":    message.String,
			"status":     status,
			"created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"requests": list})
}

// Админ: обработать заявку
func (h *PasswordRequestHandler) ProcessRequest(c *gin.Context) {
	reqID := c.Param("id")

	// Найти заявку
	var userID sql.NullInt64
	var username sql.NullString
	err := h.db.QueryRow("SELECT user_id, username FROM password_reset_requests WHERE id = ? AND status = 'open'", reqID).Scan(&userID, &username)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заявка не найдена или уже обработана"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !userID.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не найден пользователь для этой заявки"})
		return
	}

	// Проверить, что пользователь не является AD-пользователем
	var isADUser bool
	var dbUserID int
	err = h.db.QueryRow("SELECT id, is_ad_user FROM users WHERE id = ?", userID.Int64).Scan(&dbUserID, &isADUser)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пользователь не найден"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isADUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя сбросить пароль для AD-пользователя"})
		return
	}

	// Генерация временного пароля
	tempPassword, err := generateTempPassword(12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сгенерировать временный пароль"})
		return
	}

	hashed, err := database.HashPassword(tempPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Обновить пароль пользователя в базе данных
	_, err = h.db.Exec("UPDATE users SET password_hash = ?, password_reset_required = 1, updated_at = ? WHERE id = ?", hashed, time.Now(), userID.Int64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Маркировать заявку как обработанную
	adminID := c.GetInt("userID")
	_, err = h.db.Exec("UPDATE password_reset_requests SET status = 'processed', processed_by = ?, processed_at = ? WHERE id = ?", adminID, time.Now(), reqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возращаем временный пароль в ответе
	c.JSON(http.StatusOK, gin.H{"message": "Заявка обработана", "temp_password": tempPassword})
}

// Админ: отклонить заявку
func (h *PasswordRequestHandler) RejectRequest(c *gin.Context) {
	reqID := c.Param("id")

	// Найти заявку
	var status string
	err := h.db.QueryRow("SELECT status FROM password_reset_requests WHERE id = ?", reqID).Scan(&status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заявка не найдена"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if status != "open" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Только открытые заявки могут быть отклонены"})
		return
	}

	// Маркировать заявку как отклоненную
	adminID := c.GetInt("userID")
	_, err = h.db.Exec("UPDATE password_reset_requests SET status = 'rejected', processed_by = ?, processed_at = ? WHERE id = ?", adminID, time.Now(), reqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Заявка отклонена"})
}

func generateTempPassword(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// base64 encode and trim to requested length, replace '+' and '/' with letters
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > n {
		s = s[:n]
	}
	return s, nil
}
