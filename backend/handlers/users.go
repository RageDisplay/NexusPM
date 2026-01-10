package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	db *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	var rows *sql.Rows
	var err error

	if userRole == "admin" {
		rows, err = h.db.Query("SELECT id, username, role, department, is_ad_user, created_at FROM users")
	} else {
		rows, err = h.db.Query("SELECT id, username, role, department, is_ad_user, created_at FROM users WHERE department = ?", userDepartment)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type UserResponse struct {
		ID         int    `json:"id"`
		Username   string `json:"username"`
		Role       string `json:"role"`
		Department string `json:"department"`
		IsADUser   bool   `json:"is_ad_user"`
		CreatedAt  string `json:"created_at"`
	}

	users := []UserResponse{}
	for rows.Next() {
		var user UserResponse
		var department sql.NullString

		err := rows.Scan(&user.ID, &user.Username, &user.Role, &department, &user.IsADUser, &user.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if department.Valid {
			user.Department = department.String
		}

		users = append(users, user)
	}

	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Защита стартового администратора (ID = 1)
	if userID == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя изменить роль основного администратора системы"})
		return
	}

	var request struct {
		Role string `json:"role" binding:"required,oneof=user manager admin"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.db.Exec("UPDATE users SET role = ? WHERE id = ?", request.Role, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Роль пользователя изменена"})
}

func (h *UserHandler) UpdateUserDepartment(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var request struct {
		Department string `json:"department" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.db.Exec("UPDATE users SET department = ? WHERE id = ?", request.Department, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Информация об отделе обновлена"})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Нельзя удалить самого себя
	currentUserID := c.GetInt("userID")
	if userID == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя удалить свой аккаунт"})
		return
	}

	// Защита стартового администратора (ID = 1)
	if userID == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя удалить основного администратора системы"})
		return
	}

	// Проверяем существование пользователя
	var username string
	err = h.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, есть ли у пользователя задачи
	var taskCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ?", userID).Scan(&taskCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if taskCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Нельзя удалить пользователя с существующими задачами. Пожалуйста, сначала удалите его задачи.",
			"task_count": taskCount,
		})
		return
	}

	// Удаляем пользователя
	result, err := h.db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Пользователь успешно удалён",
		"deleted_user": username,
	})
}
