package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"task-management-backend/models"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	db *sql.DB
}

func NewTaskHandler(db *sql.DB) *TaskHandler {
	return &TaskHandler{db: db}
}

func (h *TaskHandler) GetTasks(c *gin.Context) {
	userID := c.GetInt("userID")
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	var rows *sql.Rows
	var err error

	switch userRole {
	case "admin":
		rows, err = h.db.Query(`
            SELECT t.id, t.title, t.description, t.progress, t.hours_per_week, t.load_per_month, 
                   t.weekly_info, t.planning, t.help_needed, t.user_id, t.project_id, 
                   t.created_at, t.updated_at, u.username, u.department, COALESCE(p.name, '') as project_name
            FROM tasks t 
            JOIN users u ON t.user_id = u.id 
            LEFT JOIN projects p ON t.project_id = p.id
            ORDER BY t.created_at DESC
        `)
	case "manager":
		rows, err = h.db.Query(`
            SELECT t.id, t.title, t.description, t.progress, t.hours_per_week, t.load_per_month, 
                   t.weekly_info, t.planning, t.help_needed, t.user_id, t.project_id, 
                   t.created_at, t.updated_at, u.username, u.department, COALESCE(p.name, '') as project_name
            FROM tasks t 
            JOIN users u ON t.user_id = u.id 
            LEFT JOIN projects p ON t.project_id = p.id
            WHERE u.department = ? 
            ORDER BY t.created_at DESC
        `, userDepartment)
	default:
		rows, err = h.db.Query(`
            SELECT t.id, t.title, t.description, t.progress, t.hours_per_week, t.load_per_month, 
                   t.weekly_info, t.planning, t.help_needed, t.user_id, t.project_id, 
                   t.created_at, t.updated_at, u.username, u.department, COALESCE(p.name, '') as project_name
            FROM tasks t 
            JOIN users u ON t.user_id = u.id 
            LEFT JOIN projects p ON t.project_id = p.id
            WHERE t.user_id = ? 
            ORDER BY t.created_at DESC
        `, userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		var task models.Task
		var department sql.NullString
		var projectID sql.NullInt64
		var projectName string

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Progress,
			&task.HoursPerWeek, &task.LoadPerMonth, &task.WeeklyInfo, &task.Planning,
			&task.HelpNeeded, &task.UserID, &projectID, &task.CreatedAt, &task.UpdatedAt,
			&task.Username, &department, &projectName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if department.Valid {
			task.Department = department.String
		}

		if projectID.Valid {
			task.ProjectID = int(projectID.Int64)
		}

		task.ProjectName = projectName
		tasks = append(tasks, task)
	}

	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID := c.GetInt("userID")
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	var task models.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Обычный пользователь может создавать только для себя
	if userRole == "user" && task.UserID != 0 && task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете создавать задачи только для себя"})
		return
	}

	// Если это не админ, то manager может создавать задачи только для своего отдела
	if userRole == "manager" && task.UserID != 0 {
		// Проверяем, что пользователь принадлежит отделу менеджера
		var taskUserDept sql.NullString
		err := h.db.QueryRow("SELECT department FROM users WHERE id = ?", task.UserID).Scan(&taskUserDept)
		if err != nil || !taskUserDept.Valid || taskUserDept.String != userDepartment {
			c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете назначать задачи только сотрудникам своего отдела"})
			return
		}
	}

	// Если task.UserID == 0, то задача создаётся для текущего пользователя
	assignToUserID := task.UserID
	if assignToUserID == 0 {
		assignToUserID = userID
	}

	result, err := h.db.Exec(`
        INSERT INTO tasks (title, description, progress, hours_per_week, load_per_month, weekly_info, planning, help_needed, user_id, project_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, task.Title, task.Description, task.Progress, task.HoursPerWeek, task.LoadPerMonth, task.WeeklyInfo, task.Planning, task.HelpNeeded, assignToUserID, sql.NullInt64{Int64: int64(task.ProjectID), Valid: task.ProjectID > 0})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	task.ID = int(id)
	task.UserID = assignToUserID
	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskID, _ := strconv.Atoi(c.Param("id"))
	userID := c.GetInt("userID")
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	// Проверка прав доступа и получение текущих данных задачи
	var taskUserID int
	var taskUserDept sql.NullString
	var currentTitle, currentDescription string
	var currentProjectID sql.NullInt64

	err := h.db.QueryRow(`
		SELECT t.user_id, u.department, t.title, t.description, t.project_id
		FROM tasks t
		LEFT JOIN users u ON t.user_id = u.id
		WHERE t.id = ?
	`, taskID).Scan(&taskUserID, &taskUserDept, &currentTitle, &currentDescription, &currentProjectID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Задача не найдена"})
		return
	}

	if userRole == "user" && taskUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "В доступе отказано"})
		return
	}

	if userRole == "manager" && (!taskUserDept.Valid || taskUserDept.String != userDepartment) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете редактировать только задачи вашего отдела"})
		return
	}

	var task models.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Обычный пользователь не может менять название, описание и проект
	if userRole == "user" {
		task.Title = currentTitle
		task.Description = currentDescription
		if currentProjectID.Valid {
			task.ProjectID = int(currentProjectID.Int64)
		} else {
			task.ProjectID = 0
		}
	}

	// Валидация данных
	if task.Progress < 0 || task.Progress > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Прогресс может быть в промежутке от 0 до 100"})
		return
	}
	if task.LoadPerMonth < 0 || task.LoadPerMonth > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нагрузка может быть в промежутке от 0 до 100"})
		return
	}
	if task.HoursPerWeek < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Часы не могут быть отрицательными"})
		return
	}

	_, err = h.db.Exec(`
        UPDATE tasks 
        SET title = ?, description = ?, progress = ?, hours_per_week = ?, load_per_month = ?, weekly_info = ?, planning = ?, help_needed = ?, project_id = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `, task.Title, task.Description, task.Progress, task.HoursPerWeek, task.LoadPerMonth, task.WeeklyInfo, task.Planning, task.HelpNeeded, sql.NullInt64{Int64: int64(task.ProjectID), Valid: task.ProjectID > 0}, taskID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Задача успешно обновлена"})
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID, _ := strconv.Atoi(c.Param("id"))
	userID := c.GetInt("userID")
	userRole := c.GetString("userRole")
	userDepartment := c.GetString("userDepartment")

	// Проверка прав доступа
	var taskUserID int
	var taskUserDept sql.NullString
	err := h.db.QueryRow("SELECT user_id, (SELECT department FROM users WHERE id = tasks.user_id) FROM tasks WHERE id = ?", taskID).Scan(&taskUserID, &taskUserDept)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Задача не найдена"})
		return
	}

	if userRole == "user" && taskUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "В доступе отказано"})
		return
	}

	if userRole == "manager" && (!taskUserDept.Valid || taskUserDept.String != userDepartment) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Вы можете удалять только задачи вашего отдела"})
		return
	}

	_, err = h.db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Задача успешно удалена"})
}
