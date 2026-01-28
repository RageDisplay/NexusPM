package database

import (
	"database/sql"
	"log"
)

// GetManagerDepartments возвращает список всех отделов, к которым имеет доступ менеджер
// Включает его основной отдел и дополнительные отделы
func GetManagerDepartments(db *sql.DB, managerID int) ([]string, error) {
	var departments []string

	// Получаем основной отдел менеджера
	var mainDept sql.NullString
	err := db.QueryRow("SELECT department FROM users WHERE id = ? AND role = 'manager'", managerID).Scan(&mainDept)
	if err != nil {
		log.Printf("Error getting manager department for %d: %v", managerID, err)
		return nil, err
	}

	// Добавляем основной отдел
	if mainDept.Valid && mainDept.String != "" {
		departments = append(departments, mainDept.String)
		log.Printf("Manager %d main department: %s", managerID, mainDept.String)
	}

	// Получаем дополнительные отделы
	rows, err := db.Query("SELECT DISTINCT department FROM manager_departments WHERE manager_id = ? ORDER BY department", managerID)
	if err != nil {
		log.Printf("Error querying manager departments for %d: %v", managerID, err)
		return departments, err
	}
	defer rows.Close()

	for rows.Next() {
		var dept string
		if err := rows.Scan(&dept); err != nil {
			log.Printf("Error scanning department: %v", err)
			continue
		}
		// Проверяем, чтобы не добавить основной отдел дважды
		isDuplicate := false
		for _, d := range departments {
			if d == dept {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			departments = append(departments, dept)
			log.Printf("Manager %d additional department: %s", managerID, dept)
		}
	}

	log.Printf("Manager %d final departments list: %v", managerID, departments)
	return departments, nil
}

// AddDepartmentAccess добавляет дополнительный отдел для менеджера
func AddDepartmentAccess(db *sql.DB, managerID int, department string, grantedByID int) error {
	log.Printf("Adding department '%s' for manager %d", department, managerID)

	// Проверяем, что это действительно менеджер
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE id = ?", managerID).Scan(&role)
	if err != nil || role != "manager" {
		log.Printf("Error: Manager %d not found or not a manager", managerID)
		return err
	}

	// Проверяем, что это не основной отдел менеджера
	var mainDept sql.NullString
	err = db.QueryRow("SELECT department FROM users WHERE id = ?", managerID).Scan(&mainDept)
	if err != nil {
		log.Printf("Error getting manager %d department: %v", managerID, err)
		return err
	}

	if mainDept.Valid && mainDept.String == department {
		// Это уже основной отдел менеджера, добавлять не нужно
		log.Printf("Department '%s' is already main department for manager %d", department, managerID)
		return nil
	}

	// Добавляем доступ к дополнительному отделу
	log.Printf("Inserting into manager_departments: manager_id=%d, department='%s'", managerID, department)
	_, err = db.Exec(`
		INSERT OR IGNORE INTO manager_departments (manager_id, department, granted_by)
		VALUES (?, ?, ?)
	`, managerID, department, grantedByID)

	if err != nil {
		log.Printf("Error inserting department access: %v", err)
	} else {
		log.Printf("Successfully added department '%s' for manager %d", department, managerID)
	}
	return err
}

// RemoveDepartmentAccess удаляет доступ менеджера к дополнительному отделу
func RemoveDepartmentAccess(db *sql.DB, managerID int, department string) error {
	// Проверяем, что это не основной отдел менеджера
	var mainDept sql.NullString
	err := db.QueryRow("SELECT department FROM users WHERE id = ?", managerID).Scan(&mainDept)
	if err != nil {
		return err
	}

	if mainDept.Valid && mainDept.String == department {
		// Нельзя удалить основной отдел менеджера
		return nil
	}

	// Удаляем доступ к дополнительному отделу
	_, err = db.Exec(`
		DELETE FROM manager_departments
		WHERE manager_id = ? AND department = ?
	`, managerID, department)

	return err
}

// GetAllDepartments возвращает список всех существующих отделов в системе
func GetAllDepartments(db *sql.DB) ([]string, error) {
	var departments []string

	rows, err := db.Query("SELECT DISTINCT department FROM users WHERE department IS NOT NULL AND department != '' ORDER BY department")
	if err != nil {
		log.Printf("Error querying departments: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dept string
		if err := rows.Scan(&dept); err != nil {
			log.Printf("Error scanning department: %v", err)
			continue
		}
		departments = append(departments, dept)
	}

	log.Printf("GetAllDepartments returned: %v", departments)
	return departments, nil
}

// GetManagerAdditionalDepartments возвращает только дополнительные отделы менеджера (без основного)
func GetManagerAdditionalDepartments(db *sql.DB, managerID int) ([]string, error) {
	var departments []string

	rows, err := db.Query("SELECT DISTINCT department FROM manager_departments WHERE manager_id = ? ORDER BY department", managerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dept string
		if err := rows.Scan(&dept); err != nil {
			log.Printf("Error scanning department: %v", err)
			continue
		}
		departments = append(departments, dept)
	}

	return departments, nil
}
