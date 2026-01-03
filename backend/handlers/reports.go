package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ReportHandler struct {
	db *sql.DB
}

func NewReportHandler(db *sql.DB) *ReportHandler {
	return &ReportHandler{db: db}
}

func (h *ReportHandler) ExportMyTasks(c *gin.Context) {
	userID := c.GetInt("userID")

	rows, err := h.db.Query(`
        SELECT title, progress, hours_per_week, load_per_month, weekly_info, planning, help_needed, created_at
        FROM tasks WHERE user_id = ?
    `, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Мои задачи")

	// Заголовки согласно формату таблицы
	headers := []string{"Проекты", "Информация за неделю", "Планирование", "Требуется помощь", "Загрузка на неделю, ч", "Загрузка на месяц, %", "Прогресс (%)"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Мои задачи", cell, header)
	}

	// Данные
	rowIndex := 2
	for rows.Next() {
		var title, weeklyInfo, planning, helpNeeded string
		var progress, loadPerMonth int
		var hoursPerWeek float64
		var createdAt time.Time

		err := rows.Scan(&title, &progress, &hoursPerWeek, &loadPerMonth, &weeklyInfo, &planning, &helpNeeded, &createdAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		data := []interface{}{title, weeklyInfo, planning, helpNeeded, hoursPerWeek, loadPerMonth, progress}
		for i, value := range data {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowIndex)
			f.SetCellValue("Мои задачи", cell, value)
		}
		rowIndex++
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=my_tasks.xlsx")
	f.Write(c.Writer)
}

func (h *ReportHandler) ExportDepartmentTasks(c *gin.Context) {
	userDepartment := c.GetString("userDepartment")

	rows, err := h.db.Query(`
        SELECT ROW_NUMBER() OVER (ORDER BY t.created_at) as row_num, u.username, t.title, t.weekly_info, t.planning, t.help_needed, t.hours_per_week, t.load_per_month
        FROM tasks t 
        JOIN users u ON t.user_id = u.id 
        WHERE u.department = ?
        ORDER BY t.created_at DESC`, userDepartment)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Статистика по отделу")

	// Заголовки согласно формату таблицы
	headers := []string{"№", "ФИО", "Проекты", "Информация за неделю", "Планирование", "Требуется помощь", "Загрузка на неделю, ч", "Загрузка на месяц, %"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Статистика по отделу", cell, header)
	}

	rowIndex := 2
	for rows.Next() {
		var rowNum int
		var username, title, weeklyInfo, planning, helpNeeded string
		var hoursPerWeek float64
		var loadPerMonth int

		err := rows.Scan(&rowNum, &username, &title, &weeklyInfo, &planning, &helpNeeded, &hoursPerWeek, &loadPerMonth)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		data := []interface{}{rowNum, username, title, weeklyInfo, planning, helpNeeded, hoursPerWeek, loadPerMonth}
		for i, value := range data {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowIndex)
			f.SetCellValue("Статистика по отделу", cell, value)
		}
		rowIndex++
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=department_statistics.xlsx")
	f.Write(c.Writer)
}

func (h *ReportHandler) ExportAllTasks(c *gin.Context) {
	rows, err := h.db.Query(`
        SELECT ROW_NUMBER() OVER (ORDER BY t.created_at) as row_num, u.username, t.title, t.weekly_info, t.planning, t.help_needed, t.hours_per_week, t.load_per_month
        FROM tasks t 
        JOIN users u ON t.user_id = u.id
        ORDER BY t.created_at DESC`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Статистика по проектам")

	// Заголовки согласно формату таблицы
	headers := []string{"№", "ФИО", "Проекты", "Информация за неделю", "Планирование", "Требуется помощь", "Загрузка на неделю, ч", "Загрузка на месяц, %"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Статистика по проектам", cell, header)
	}

	rowIndex := 2
	for rows.Next() {
		var rowNum int
		var username, title, weeklyInfo, planning, helpNeeded string
		var hoursPerWeek float64
		var loadPerMonth int

		err := rows.Scan(&rowNum, &username, &title, &weeklyInfo, &planning, &helpNeeded, &hoursPerWeek, &loadPerMonth)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		data := []interface{}{rowNum, username, title, weeklyInfo, planning, helpNeeded, hoursPerWeek, loadPerMonth}
		for i, value := range data {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowIndex)
			f.SetCellValue("Статистика по проектам", cell, value)
		}
		rowIndex++
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=statistics_by_projects.xlsx")
	f.Write(c.Writer)
}
