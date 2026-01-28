package handlers

import (
	"database/sql"
	"fmt"
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

	// Получаем параметры дат
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	query := `
        SELECT title, progress, hours_per_week, load_per_month, weekly_info, planning, help_needed, created_at
        FROM tasks WHERE user_id = ?`

	args := []interface{}{userID}

	// Добавляем фильтр по датам если указаны
	if startDateStr != "" && endDateStr != "" {
		query += ` AND created_at >= ? AND created_at <= ?`
		args = append(args, startDateStr+" 00:00:00", endDateStr+" 23:59:59")
	}

	query += ` ORDER BY created_at DESC`

	rows, err := h.db.Query(query, args...)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheet := "Мои задачи"
	f.SetSheetName("Sheet1", sheet)

	// Заголовки согласно формату таблицы
	headers := []string{"Проекты", "Информация за неделю", "Планирование", "Требуется помощь", "Загрузка на неделю, ч", "Загрузка на месяц, %", "Прогресс (%)"}

	// Слайс для хранения максимальной длины для каждой колонки
	maxLen := make([]int, len(headers))

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
		if l := len(header); l > maxLen[i] {
			maxLen[i] = l
		}
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
			f.SetCellValue(sheet, cell, value)

			// Обновляем maxLen для колонки
			var s string
			switch v := value.(type) {
			case string:
				s = v
			default:
				s = fmt.Sprintf("%v", v)
			}
			if l := len(s); l > maxLen[i] {
				maxLen[i] = l
			}
		}
		rowIndex++
	}

	// Устанавливаем автофильтр (через создание таблицы) и ширину колонок
	lastCol := colLetter(len(headers))
	tableRange := fmt.Sprintf("A1:%s%d", lastCol, rowIndex-1)
	// Создаём таблицу Excel — это добавит фильтры на заголовки
	tbl := &excelize.Table{
		Name:      "Table_MyTasks",
		Range:     tableRange,
		StyleName: "TableStyleMedium2",
	}
	_ = f.AddTable(sheet, tbl)

	// Стиль для переноса текста
	wrapStyleID, _ := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true}})
	// Стиль для заголовка (жирный)
	headerStyleID, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	_ = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", lastCol), headerStyleID)
	// Применяем перенос к всему диапазону данных
	_ = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s%d", lastCol, rowIndex-1), wrapStyleID)

	// Устанавливаем ширину колонок, пропорционально максимальной длине
	for i := range headers {
		col := colLetter(i + 1)
		width := float64(maxLen[i]) * 1.2
		if width < 10 {
			width = 10
		}
		if width > 60 {
			width = 60
		}
		_ = f.SetColWidth(sheet, col, col, width)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=my_tasks.xlsx")
	f.Write(c.Writer)
}

func (h *ReportHandler) ExportDepartmentTasks(c *gin.Context) {
	userDepartment := c.GetString("userDepartment")

	// Получаем параметры дат
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	query := `
        SELECT ROW_NUMBER() OVER (ORDER BY t.created_at) as row_num, u.username, t.title, t.weekly_info, t.planning, t.help_needed, t.hours_per_week, t.load_per_month
        FROM tasks t 
        JOIN users u ON t.user_id = u.id 
        WHERE u.department = ?`

	args := []interface{}{userDepartment}

	// Добавляем фильтр по датам если указаны
	if startDateStr != "" && endDateStr != "" {
		query += ` AND t.created_at >= ? AND t.created_at <= ?`
		args = append(args, startDateStr+" 00:00:00", endDateStr+" 23:59:59")
	}

	query += ` ORDER BY t.created_at DESC`

	rows, err := h.db.Query(query, args...)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheet := "Статистика по отделу"
	f.SetSheetName("Sheet1", sheet)

	// Заголовки согласно формату таблицы
	headers := []string{"№", "ФИО", "Проекты", "Информация за неделю", "Планирование", "Требуется помощь", "Загрузка на неделю, ч", "Загрузка на месяц, %"}
	maxLen := make([]int, len(headers))
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
		if l := len(header); l > maxLen[i] {
			maxLen[i] = l
		}
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
			f.SetCellValue(sheet, cell, value)

			var s string
			switch v := value.(type) {
			case string:
				s = v
			default:
				s = fmt.Sprintf("%v", v)
			}
			if l := len(s); l > maxLen[i] {
				maxLen[i] = l
			}
		}
		rowIndex++
	}

	lastCol := colLetter(len(headers))
	tableRange := fmt.Sprintf("A1:%s%d", lastCol, rowIndex-1)
	tbl := &excelize.Table{
		Name:      "Table_Department",
		Range:     tableRange,
		StyleName: "TableStyleMedium2",
	}
	_ = f.AddTable(sheet, tbl)
	wrapStyleID, _ := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true}})
	headerStyleID, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	_ = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", lastCol), headerStyleID)
	_ = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s%d", lastCol, rowIndex-1), wrapStyleID)

	for i := range headers {
		col := colLetter(i + 1)
		width := float64(maxLen[i]) * 1.2
		if width < 10 {
			width = 10
		}
		if width > 60 {
			width = 60
		}
		_ = f.SetColWidth(sheet, col, col, width)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=department_statistics.xlsx")
	f.Write(c.Writer)
}

func (h *ReportHandler) ExportAllTasks(c *gin.Context) {
	// Получаем параметры дат
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	query := `
        SELECT ROW_NUMBER() OVER (ORDER BY t.created_at) as row_num, u.username, t.title, t.weekly_info, t.planning, t.help_needed, t.hours_per_week, t.load_per_month
        FROM tasks t 
        JOIN users u ON t.user_id = u.id`

	args := []interface{}{}

	// Добавляем фильтр по датам если указаны
	if startDateStr != "" && endDateStr != "" {
		query += ` WHERE t.created_at >= ? AND t.created_at <= ?`
		args = append(args, startDateStr+" 00:00:00", endDateStr+" 23:59:59")
	}

	query += ` ORDER BY t.created_at DESC`

	rows, err := h.db.Query(query, args...)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheet := "Статистика по проектам"
	f.SetSheetName("Sheet1", sheet)

	// Заголовки согласно формату таблицы
	headers := []string{"№", "ФИО", "Проекты", "Информация за неделю", "Планирование", "Требуется помощь", "Загрузка на неделю, ч", "Загрузка на месяц, %"}
	maxLen := make([]int, len(headers))
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
		if l := len(header); l > maxLen[i] {
			maxLen[i] = l
		}
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
			f.SetCellValue(sheet, cell, value)

			var s string
			switch v := value.(type) {
			case string:
				s = v
			default:
				s = fmt.Sprintf("%v", v)
			}
			if l := len(s); l > maxLen[i] {
				maxLen[i] = l
			}
		}
		rowIndex++
	}

	lastCol := colLetter(len(headers))
	tableRange := fmt.Sprintf("A1:%s%d", lastCol, rowIndex-1)
	tbl := &excelize.Table{
		Name:      "Table_AllTasks",
		Range:     tableRange,
		StyleName: "TableStyleMedium2",
	}
	_ = f.AddTable(sheet, tbl)
	wrapStyleID, _ := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true}})
	headerStyleID, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	_ = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", lastCol), headerStyleID)
	_ = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s%d", lastCol, rowIndex-1), wrapStyleID)

	for i := range headers {
		col := colLetter(i + 1)
		width := float64(maxLen[i]) * 1.2
		if width < 10 {
			width = 10
		}
		if width > 60 {
			width = 60
		}
		_ = f.SetColWidth(sheet, col, col, width)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=statistics_by_projects.xlsx")
	f.Write(c.Writer)
}

// colLetter возвращает букву колонки Excel по её номеру (1 -> A, 27 -> AA)
func colLetter(n int) string {
	letters := ""
	for n > 0 {
		n--
		letters = string(rune('A'+(n%26))) + letters
		n = n / 26
	}
	return letters
}
