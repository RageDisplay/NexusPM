package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"task-management-backend/database"
	"task-management-backend/models"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	db *sql.DB
}

func NewProjectHandler(db *sql.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

// GetProjects получить все проекты (менеджеры видят ВСЕ проекты и могут назначать сотрудников)
func (h *ProjectHandler) GetProjects(c *gin.Context) {
	userRole := c.GetString("userRole")

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	// Менеджеры и админы видят ВСЕ проекты
	rows, err := h.db.Query(`
        SELECT id, name, description, created_by, department, created_at, updated_at
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
		err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedBy, &project.Department, &project.CreatedAt, &project.UpdatedAt)
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
        SELECT DISTINCT p.id, p.name, p.description, p.created_by, p.department, p.created_at, p.updated_at
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
		err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedBy, &project.Department, &project.CreatedAt, &project.UpdatedAt)
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
	userDepartment := c.GetString("userDepartment")

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для создания проектов"})
		return
	}

	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Для менеджера department устанавливается автоматически (его отдел)
	// Для админа можно указать в request
	if userRole == "manager" {
		project.Department = userDepartment
	} else if project.Department == "" {
		// Админ должен указать отдел
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пожалуйста укажите отдел для проекта"})
		return
	}

	result, err := h.db.Exec(`
        INSERT INTO projects (name, description, created_by, department)
        VALUES (?, ?, ?, ?)
    `, project.Name, project.Description, userID, project.Department)

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
// Менеджер может назначать пользователей из всех доступных ему отделов на любые проекты
// Админ может назначать кого угодно на любой проект
func (h *ProjectHandler) AssignProjectToUser(c *gin.Context) {
	userID := c.GetInt("userID")
	userRole := c.GetString("userRole")

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

	// Проверяем права менеджера
	if userRole == "manager" {
		// Менеджер может назначать пользователей только из доступных ему отделов
		var targetUserDept sql.NullString
		err := h.db.QueryRow("SELECT department FROM users WHERE id = ?", req.UserID).Scan(&targetUserDept)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке пользователя"})
			return
		}

		targetDept := ""
		if targetUserDept.Valid {
			targetDept = targetUserDept.String
		}

		// Получаем все доступные отделы менеджера
		availableDepts, err := database.GetManagerDepartments(h.db, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Проверяем, что пользователь из одного из доступных отделов
		isAllowed := false
		for _, dept := range availableDepts {
			if dept == targetDept {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете назначать проекты только пользователям из доступных вам отделов"})
			return
		}
	}

	_, err := h.db.Exec(`
        INSERT OR IGNORE INTO user_projects (user_id, project_id)
        VALUES (?, ?)
    `, req.UserID, req.ProjectID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Получаем название проекта для уведомления
	var projectName string
	h.db.QueryRow("SELECT name FROM projects WHERE id = ?", req.ProjectID).Scan(&projectName)

	// Создаем уведомление для пользователя о назначении проекта
	CreateNotification(h.db, req.UserID, "project_assigned", "Назначен новый проект", "Вам назначен проект: "+projectName, &req.ProjectID)

	c.JSON(http.StatusCreated, gin.H{"message": "Проект успешно назначен"})
}

// RemoveProjectFromUser удалить проект у пользователя (для админа и менеджера)
// Менеджер может удалять проекты только у пользователей из доступных ему отделов
// Админ может удалять кого угодно со всех проектов
func (h *ProjectHandler) RemoveProjectFromUser(c *gin.Context) {
	userID := c.GetInt("userID")
	userRole := c.GetString("userRole")

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

	// Проверяем права менеджера
	if userRole == "manager" {
		// Менеджер может удалять проекты только у пользователей из доступных ему отделов
		var targetUserDept sql.NullString
		err := h.db.QueryRow("SELECT department FROM users WHERE id = ?", req.UserID).Scan(&targetUserDept)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке пользователя"})
			return
		}

		targetDept := ""
		if targetUserDept.Valid {
			targetDept = targetUserDept.String
		}

		// Получаем все доступные отделы менеджера
		availableDepts, err := database.GetManagerDepartments(h.db, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Проверяем, что пользователь из одного из доступных отделов
		isAllowed := false
		for _, dept := range availableDepts {
			if dept == targetDept {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете удалять проекты только у пользователей из доступных вам отделов"})
			return
		}
	}

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
        SELECT id, name, description, created_by, department, created_at, updated_at
        FROM projects
        WHERE id = ?
    `, projectID).Scan(&project.ID, &project.Name, &project.Description, &project.CreatedBy, &project.Department, &project.CreatedAt, &project.UpdatedAt)

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
// Менеджер может видеть пользователей всех проектов, но управлять только своих
func (h *ProjectHandler) GetProjectUsers(c *gin.Context) {
	userRole := c.GetString("userRole")
	projectID, _ := strconv.Atoi(c.Param("id"))

	if userRole != "admin" && userRole != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	// Менеджер может видеть пользователей всех проектов (но управлять может только своими)
	// Но для доступности можем проверить что проект существует
	var projectExists int
	err := h.db.QueryRow("SELECT COUNT(*) FROM projects WHERE id = ?", projectID).Scan(&projectExists)
	if err != nil || projectExists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Проект не найден"})
		return
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

// GetUserProjectsWithStats получить проекты пользователя со статистикой (дата последнего отчета и статус)
func (h *ProjectHandler) GetUserProjectsWithStats(c *gin.Context) {
	userID := c.GetInt("userID")

	// Логируем для отладки
	println("[DEBUG] GetUserProjectsWithStats called for userID:", userID)

	// Получаем все проекты пользователя (его собственные + назначенные)
	rows, err := h.db.Query(`
        SELECT DISTINCT p.id, p.name, p.description, p.created_by, p.department, p.created_at, p.updated_at
        FROM projects p
        LEFT JOIN user_projects up ON p.id = up.project_id
        WHERE up.user_id = ? OR p.created_by = ?
        ORDER BY p.name
    `, userID, userID)
	if err != nil {
		println("[ERROR] Failed to query projects:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type ProjectWithStats struct {
		models.Project
		LastReportDate *string `json:"last_report_date"`
	}

	projects := []ProjectWithStats{}
	for rows.Next() {
		var project ProjectWithStats
		err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedBy,
			&project.Department, &project.CreatedAt, &project.UpdatedAt)
		if err != nil {
			println("[ERROR] Failed to scan project:", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Получаем дату последнего отчета для этого проекта (только для текущего пользователя)
		var lastReportDate sql.NullTime
		err = h.db.QueryRow(`
            SELECT MAX(created_at) FROM tasks WHERE project_id = ? AND user_id = ?
        `, project.ID, userID).Scan(&lastReportDate)

		if err != nil && err != sql.ErrNoRows {
			println("[ERROR] Failed to get last report date for project", project.ID, ":", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if lastReportDate.Valid {
			dateStr := lastReportDate.Time.Format("2006-01-02T15:04:05Z07:00")
			project.LastReportDate = &dateStr
			println("[DEBUG] Project", project.ID, "has last_report_date:", dateStr)
		} else {
			println("[DEBUG] Project", project.ID, "has no reports")
		}
		projects = append(projects, project)
	}

	println("[DEBUG] Returning", len(projects), "projects")
	c.JSON(http.StatusOK, projects)
}

// DeleteProject удалить проект (для админа и менеджера, создавшего проект)
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	userRole := c.GetString("userRole")
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
		// Получаем текущего пользователя из контекста
		userID := c.GetInt("userID")
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
