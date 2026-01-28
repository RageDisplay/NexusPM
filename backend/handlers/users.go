package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"task-management-backend/database"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	db *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	userID := c.GetInt("userID")
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	var rows *sql.Rows
	var err error

	if userRole == "admin" {
		rows, err = h.db.Query("SELECT id, username, role, department, is_ad_user, created_at FROM users")
	} else if userRole == "manager" {
		// Получаем все доступные отделы для менеджера (основной + дополнительные)
		departments, err := database.GetManagerDepartments(h.db, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Если менеджер не имеет никаких отделов, показываем только его основной
		if len(departments) == 0 {
			departments = []string{userDepartment}
		}

		log.Printf("Manager %d has departments: %v", userID, departments)

		// Строим запрос с фильтром по всем доступным отделам
		query := "SELECT id, username, role, department, is_ad_user, created_at FROM users WHERE department IN ("
		args := make([]interface{}, len(departments))
		for i, dept := range departments {
			args[i] = dept
			if i > 0 {
				query += ", "
			}
			query += "?"
		}
		query += ")"

		log.Printf("Query: %s with args: %v", query, args)
		rows, err = h.db.Query(query, args...)
	} else {
		rows, err = h.db.Query("SELECT id, username, role, department, is_ad_user, created_at FROM users WHERE department = ?", userDepartment)
	}

	if err != nil {
		log.Printf("Query execution error: %v", err)
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
	rowCount := 0
	for rows.Next() {
		rowCount++
		var user UserResponse
		var department sql.NullString

		err := rows.Scan(&user.ID, &user.Username, &user.Role, &department, &user.IsADUser, &user.CreatedAt)
		if err != nil {
			log.Printf("Scan error at row %d: %v", rowCount, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if department.Valid {
			user.Department = department.String
		}

		users = append(users, user)
		log.Printf("  Row %d: %s (%s) in dept: %s", rowCount, user.Username, user.Role, user.Department)
	}

	log.Printf("GetUsers returning %d users for %s role user %d", len(users), userRole, userID)

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

// GetManagerDepartments возвращает список всех отделов, к которым имеет доступ менеджер
func (h *UserHandler) GetManagerDepartments(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	departments, err := database.GetManagerDepartments(h.db, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, departments)
}

// GetManagerAdditionalDepartments возвращает только дополнительные отделы менеджера
func (h *UserHandler) GetManagerAdditionalDepartments(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	departments, err := database.GetManagerAdditionalDepartments(h.db, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, departments)
}

// AddDepartmentAccess добавляет дополнительный отдел для менеджера (только для админа)
func (h *UserHandler) AddDepartmentAccess(c *gin.Context) {
	managerID, err := strconv.Atoi(c.Param("id"))
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

	// Получаем ID текущего администратора
	adminID := c.GetInt("userID")

	// Добавляем доступ к отделу
	err = database.AddDepartmentAccess(h.db, managerID, request.Department, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Доступ к отделу успешно добавлен"})
}

// RemoveDepartmentAccess удаляет доступ менеджера к дополнительному отделу (только для админа)
func (h *UserHandler) RemoveDepartmentAccess(c *gin.Context) {
	managerID, err := strconv.Atoi(c.Param("id"))
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

	// Удаляем доступ к отделу
	err = database.RemoveDepartmentAccess(h.db, managerID, request.Department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Доступ к отделу успешно удалён"})
}

// GetAllDepartments возвращает список всех отделов в системе
func (h *UserHandler) GetAllDepartments(c *gin.Context) {
	departments, err := database.GetAllDepartments(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, departments)
}
