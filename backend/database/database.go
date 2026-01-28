package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	// Создаем директорию для данных, если её нет
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "tasks.db")
	log.Printf("Initializing database at: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Создание таблицы пользователей
	createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username VARCHAR(50) UNIQUE NOT NULL,
        password_hash TEXT,
        role VARCHAR(20) NOT NULL DEFAULT 'user',
        department VARCHAR(100),
        first_name VARCHAR(100),
        last_name VARCHAR(100),
        patronymic VARCHAR(100),
        is_ad_user BOOLEAN DEFAULT 0,
        ad_user_id VARCHAR(255),
        password_reset_required BOOLEAN DEFAULT 0,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	// Таблица конфигурации AD
	createADConfigTable := `
    CREATE TABLE IF NOT EXISTS ad_config (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        enabled BOOLEAN DEFAULT 0,
        directory_type VARCHAR(50) NOT NULL,
        server_url VARCHAR(255) NOT NULL,
        base_dn VARCHAR(255) NOT NULL,
        bind_dn VARCHAR(255) NOT NULL,
        bind_password TEXT NOT NULL,
        user_search_base VARCHAR(255) NOT NULL,
        user_name_attr VARCHAR(100) NOT NULL,
        department_attr VARCHAR(100),
        email_attr VARCHAR(100),
        group_search_base VARCHAR(255),
        sync_interval INTEGER DEFAULT 60,
        tls_enabled BOOLEAN DEFAULT 0,
        certificate_path VARCHAR(500),
        skip_cert_verify BOOLEAN DEFAULT 0,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	// Создание таблицы проектов
	createProjectsTable := `
    CREATE TABLE IF NOT EXISTS projects (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name VARCHAR(255) NOT NULL UNIQUE,
        description TEXT,
        created_by INTEGER NOT NULL,
        department VARCHAR(100) NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (created_by) REFERENCES users (id)
    );`

	// Таблица связи пользователей и проектов
	createUserProjectsTable := `
    CREATE TABLE IF NOT EXISTS user_projects (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        project_id INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(user_id, project_id),
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (project_id) REFERENCES projects (id)
    );`

	// Создание таблицы задач
	createTasksTable := `
    CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        progress INTEGER DEFAULT 0,
        hours_per_week DECIMAL(10,2) DEFAULT 0,
        load_per_month INTEGER DEFAULT 0,
        weekly_info TEXT,
        planning TEXT,
        help_needed TEXT,
        user_id INTEGER NOT NULL,
        project_id INTEGER,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (project_id) REFERENCES projects (id)
    );`

	tables := []string{createUsersTable, createProjectsTable, createUserProjectsTable, createTasksTable, createADConfigTable}

	// Таблица заявок на сброс пароля
	createPasswordRequestsTable := `
    CREATE TABLE IF NOT EXISTS password_reset_requests (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        username VARCHAR(50),
        message TEXT,
        status VARCHAR(20) DEFAULT 'open',
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        processed_by INTEGER,
        processed_at DATETIME,
        FOREIGN KEY (user_id) REFERENCES users (id)
    );`

	// Таблица активностей пользователя
	createActivityTable := `
    CREATE TABLE IF NOT EXISTS activity_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        action_type VARCHAR(50) NOT NULL,
        description TEXT,
        task_id INTEGER,
        project_id INTEGER,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (task_id) REFERENCES tasks (id),
        FOREIGN KEY (project_id) REFERENCES projects (id)
    );`

	// Таблица уведомлений
	createNotificationsTable := `
    CREATE TABLE IF NOT EXISTS notifications (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        notification_type VARCHAR(50) NOT NULL,
        title TEXT NOT NULL,
        message TEXT,
        related_id INTEGER,
        is_read BOOLEAN DEFAULT 0,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id)
    );`

	// Таблица чёрного списка токенов
	createTokenBlacklistTable := `
    CREATE TABLE IF NOT EXISTS token_blacklist (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        token_hash VARCHAR(255) UNIQUE NOT NULL,
        expires_at DATETIME NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id)
    );`

	// Таблица дополнительных отделов для менеджеров
	createManagerDepartmentsTable := `
    CREATE TABLE IF NOT EXISTS manager_departments (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        manager_id INTEGER NOT NULL,
        department VARCHAR(100) NOT NULL,
        granted_by INTEGER,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(manager_id, department),
        FOREIGN KEY (manager_id) REFERENCES users (id),
        FOREIGN KEY (granted_by) REFERENCES users (id)
    );`

	tables = append(tables, createPasswordRequestsTable, createActivityTable, createNotificationsTable, createTokenBlacklistTable, createManagerDepartmentsTable)
	for _, table := range tables {
		_, err = db.Exec(table)
		if err != nil {
			return nil, err
		}
	}

	// Создание администратора по умолчанию
	hashedPassword, _ := HashPassword("main12!@") // Поменять после инициализации системы
	db.Exec(`INSERT OR IGNORE INTO users (username, password_hash, role, department, is_ad_user, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, "admin", hashedPassword, "admin", "Администрация", false, time.Now(), time.Now())

	log.Println("Database initialized successfully")
	return db, nil
}
