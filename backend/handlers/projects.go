package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"task-management-backend/models"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	db *sql.DB
}

func NewProjectHandler(db *sql.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

// GetProjects получить все проекты (только для админа/менеджера)
func (h *ProjectHandler) GetProjects(c *gin.Context) {
	userRole := c.GetString("userRole")

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	rows, err := h.db.Query(`
        SELECT id, name, description, created_by, created_at, updated_at
        FROM projects
        ORDER BY name
    `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	projects := []models.Project{}
	for rows.Next() {
		var project models.Project
		err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedBy, &project.CreatedAt, &project.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		projects = append(projects, project)
	}

	c.JSON(http.StatusOK, projects)
}

// GetUserProjects получить проекты текущего пользователя (созданные им или назначенные)
func (h *ProjectHandler) GetUserProjects(c *gin.Context) {
	userID := c.GetInt("userID")

	rows, err := h.db.Query(`
        SELECT DISTINCT p.id, p.name, p.description, p.created_by, p.created_at, p.updated_at
        FROM projects p
        LEFT JOIN user_projects up ON p.id = up.project_id
        WHERE up.user_id = ? OR p.created_by = ?
        ORDER BY p.name
    `, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	projects := []models.Project{}
	for rows.Next() {
		var project models.Project
		err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedBy, &project.CreatedAt, &project.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		projects = append(projects, project)
	}

	c.JSON(http.StatusOK, projects)
}

// CreateProject создать новый проект (для админа и менеджера)
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userRole := c.GetString("userRole")
	userID := c.GetInt("userID")

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для создания проектов"})
		return
	}

	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.Exec(`
        INSERT INTO projects (name, description, created_by)
        VALUES (?, ?, ?)
    `, project.Name, project.Description, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	project.ID = int(id)
	project.CreatedBy = userID
	c.JSON(http.StatusCreated, project)
}

// UpdateProject обновить проект (для админа и менеджера)
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	userRole := c.GetString("userRole")
	userID := c.GetInt("userID")
	projectID, _ := strconv.Atoi(c.Param("id"))

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для редактирования проектов"})
		return
	}

	// Проверяем, что менеджер редактирует только свой проект
	if userRole == "manager" {
		var createdBy int
		err := h.db.QueryRow("SELECT created_by FROM projects WHERE id = ?", projectID).Scan(&createdBy)
		if err != nil || createdBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете редактировать только свои проекты"})
			return
		}
	}

	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.Exec(`
        UPDATE projects 
        SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `, project.Name, project.Description, projectID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Проект успешно обновлён"})
}

// AssignProjectToUser назначить проект пользователю (для админа и менеджера)
func (h *ProjectHandler) AssignProjectToUser(c *gin.Context) {
	userRole := c.GetString("userRole")
	userID := c.GetInt("userID")
	userDepartment := c.GetString("userDepartment")

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для назначения проектов"})
		return
	}

	var req struct {
		UserID    int `json:"user_id" binding:"required"`
		ProjectID int `json:"project_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем права менеджера и админа
	if userRole == "manager" {
		// Менеджер может назначать только в своём отделе или самого себя
		if req.UserID != userID {
			// Если это не сам менеджер, проверяем, что целевой пользователь в его отделе
			var targetUserDept sql.NullString
			err := h.db.QueryRow("SELECT department FROM users WHERE id = ?", req.UserID).Scan(&targetUserDept)
			if err != nil || !targetUserDept.Valid || targetUserDept.String != userDepartment {
				c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете назначать проекты только сотрудникам своего отдела"})
				return
			}
		}
		// Проверяем, что проект создан менеджером
		var createdBy int
		err := h.db.QueryRow("SELECT created_by FROM projects WHERE id = ?", req.ProjectID).Scan(&createdBy)
		if err != nil || createdBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете назначать только свои проекты"})
			return
		}
	}
	// Админ может назначать кого угодно на любой проект (без дополнительных проверок)

	_, err := h.db.Exec(`
        INSERT OR IGNORE INTO user_projects (user_id, project_id)
        VALUES (?, ?)
    `, req.UserID, req.ProjectID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Проект успешно назначен"})
}

// RemoveProjectFromUser удалить проект у пользователя (для админа и менеджера)
func (h *ProjectHandler) RemoveProjectFromUser(c *gin.Context) {
	userRole := c.GetString("userRole")
	userID := c.GetInt("userID")
	userDepartment := c.GetString("userDepartment")

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для удаления проектов"})
		return
	}

	var req struct {
		UserID    int `json:"user_id" binding:"required"`
		ProjectID int `json:"project_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем права менеджера и админа
	if userRole == "manager" {
		// Менеджер может удалять только у себя и сотрудников своего отдела
		if req.UserID != userID {
			// Если это не сам менеджер, проверяем, что целевой пользователь в его отделе
			var targetUserDept sql.NullString
			err := h.db.QueryRow("SELECT department FROM users WHERE id = ?", req.UserID).Scan(&targetUserDept)
			if err != nil || !targetUserDept.Valid || targetUserDept.String != userDepartment {
				c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете удалять проекты только у сотрудников своего отдела"})
				return
			}
		}
		// Проверяем, что проект создан менеджером
		var createdBy int
		err := h.db.QueryRow("SELECT created_by FROM projects WHERE id = ?", req.ProjectID).Scan(&createdBy)
		if err != nil || createdBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете удалять только свои проекты"})
			return
		}
	}
	// Админ может удалять кого угодно со любого проекта (без дополнительных проверок)

	_, err := h.db.Exec(`
        DELETE FROM user_projects
        WHERE user_id = ? AND project_id = ?
    `, req.UserID, req.ProjectID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Проект успешно удалён"})
}

// GetProjectByID получить информацию о проекте по ID
func (h *ProjectHandler) GetProjectByID(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Param("id"))

	var project models.Project
	err := h.db.QueryRow(`
        SELECT id, name, description, created_by, created_at, updated_at
        FROM projects
        WHERE id = ?
    `, projectID).Scan(&project.ID, &project.Name, &project.Description, &project.CreatedBy, &project.CreatedAt, &project.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Проект не найден"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, project)
}

// GetProjectUsers получить пользователей проекта (для админа и менеджера)
func (h *ProjectHandler) GetProjectUsers(c *gin.Context) {
	userRole := c.GetString("userRole")
	userID := c.GetInt("userID")
	projectID, _ := strconv.Atoi(c.Param("id"))

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	// Проверяем, что менеджер видит только свои проекты
	if userRole == "manager" {
		var createdBy int
		err := h.db.QueryRow("SELECT created_by FROM projects WHERE id = ?", projectID).Scan(&createdBy)
		if err != nil || createdBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете видеть только свои проекты"})
			return
		}
	}

	rows, err := h.db.Query(`
        SELECT u.id, u.username, u.role, u.department
        FROM users u
        JOIN user_projects up ON u.id = up.user_id
        WHERE up.project_id = ?
        ORDER BY u.username
    `, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	users := []struct {
		ID         int    `json:"id"`
		Username   string `json:"username"`
		Role       string `json:"role"`
		Department string `json:"department"`
	}{}

	for rows.Next() {
		var user struct {
			ID         int    `json:"id"`
			Username   string `json:"username"`
			Role       string `json:"role"`
			Department string `json:"department"`
		}
		err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.Department)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, users)
}

// DeleteProject удалить проект (для админа и менеджера, создавшего проект)
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	userRole := c.GetString("userRole")
	userID := c.GetInt("userID")
	projectID, _ := strconv.Atoi(c.Param("id"))

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для удаления проектов"})
		return
	}

	// Получаем информацию о проекте
	var createdBy int
	err := h.db.QueryRow("SELECT created_by FROM projects WHERE id = ?", projectID).Scan(&createdBy)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Проект не найден"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем права менеджера
	if userRole == "manager" {
		// Менеджер может удалить проект только если он его создал
		if createdBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете удалять только свои проекты"})
			return
		}
	}
	// Админ может удалить любой проект

	// Удаляем все связи пользователей с этим проектом
	_, err = h.db.Exec("DELETE FROM user_projects WHERE project_id = ?", projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Удаляем сам проект
	_, err = h.db.Exec("DELETE FROM projects WHERE id = ?", projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Проект успешно удалён"})
}
