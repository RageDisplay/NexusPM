package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"task-management-backend/database"
	"task-management-backend/ldap"
	"task-management-backend/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type AuthHandler struct {
	db          *sql.DB
	ldapManager *ldap.LDAPManager
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{
		db:          db,
		ldapManager: ldap.NewLDAPManager(db),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		Department string `json:"department" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Register error - JSON bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Register attempt: username=%s, department=%s", req.Username, req.Department)

	// Проверка существования пользователя
	var exists bool
	err := h.db.QueryRow("SELECT 1 FROM users WHERE username = ?", req.Username).Scan(&exists)
	if err == nil {
		log.Printf("Register failed: user %s already exists", req.Username)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Такой пользователь уже существует"})
		return
	}

	// Хеширование пароля
	hashedPassword, err := database.HashPassword(req.Password)
	if err != nil {
		log.Printf("Register error: failed to hash password for user %s: %v", req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка формата пароля"})
		return
	}

	// Создание пользователя
	result, err := h.db.Exec(
		"INSERT INTO users (username, password_hash, role, department, is_ad_user, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Username, hashedPassword, "user", req.Department, false, time.Now(), time.Now(),
	)
	if err != nil {
		log.Printf("Register error: failed to insert user %s: %v", req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID, _ := result.LastInsertId()

	log.Printf("Register success: user %s created with id %d", req.Username, userID)

	// Генерация JWT токена
	token, err := database.GenerateJWT(int(userID), req.Username, "user", req.Department)
	if err != nil {
		log.Printf("Register error: failed to generate JWT for user %s: %v", req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка в генерации токена"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Аккаунт успешно создан",
		"token":   token,
		"user": gin.H{
			"id":         userID,
			"username":   req.Username,
			"role":       "user",
			"department": req.Department,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		AuthType string `json:"auth_type"` // "local" или "ad"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Login error - JSON bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Login attempt: username=%s, authType=%s", req.Username, req.AuthType)

	// Если не указан тип аутентификации, пытаемся оба способа
	if req.AuthType == "" {
		req.AuthType = "auto"
	}

	var user *models.User
	var token string
	var err error

	// Пытаемся аутентифицироваться через AD, если это включено
	if req.AuthType == "auto" || req.AuthType == "ad" {
		log.Printf("Attempting AD login for user: %s", req.Username)
		user, token, err = h.loginAD(req.Username, req.Password)
		if err == nil {
			log.Printf("AD login successful for user: %s", req.Username)
			h.returnLoginResponse(c, user, token)
			return
		}
		log.Printf("AD login failed for user %s: %v", req.Username, err)
		// Если явно запросили AD и он не сработал, возвращаем ошибку
		if req.AuthType == "ad" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
	}

	// Пытаемся локальную аутентификацию
	if req.AuthType == "auto" || req.AuthType == "local" {
		log.Printf("Attempting local login for user: %s", req.Username)
		user, token, err = h.loginLocal(req.Username, req.Password)
		if err == nil {
			log.Printf("Local login successful for user: %s", req.Username)
			h.returnLoginResponse(c, user, token)
			return
		}
		log.Printf("Local login failed for user %s: %v", req.Username, err)
	}

	log.Printf("All login attempts failed for user: %s", req.Username)
	c.JSON(http.StatusUnauthorized, gin.H{"error": "Неправильные логин или пароль"})
}

func (h *AuthHandler) loginAD(username, password string) (*models.User, string, error) {
	log.Printf("loginAD: attempting to authenticate user %s", username)

	// Аутентифицируемся в AD
	userInfo, err := h.ldapManager.AuthenticateADUser(username, password)
	if err != nil {
		log.Printf("loginAD: LDAP authentication failed for user %s: %v", username, err)
		return nil, "", err
	}

	// Проверяем, существует ли пользователь в нашей БД
	var user models.User

	err = h.db.QueryRow(
		"SELECT id, username, role, department, is_ad_user FROM users WHERE username = ? AND is_ad_user = 1",
		username,
	).Scan(&user.ID, &user.Username, &user.Role, &user.Department, &user.IsADUser)

	if err == sql.ErrNoRows {
		// Пользователя нет в БД, создаём его
		result, err := h.db.Exec(`
			INSERT INTO users (username, password_hash, role, department, is_ad_user, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			username,
			"", // AD users don't have password hash
			"user",
			sql.NullString{String: userInfo.Department, Valid: userInfo.Department != ""},
			true,
			time.Now(),
			time.Now(),
		)

		if err != nil {
			return nil, "", err
		}

		userID, _ := result.LastInsertId()
		user.ID = int(userID)
		user.Username = username
		user.Role = "user"
		user.IsADUser = true
		user.Department = sql.NullString{String: userInfo.Department, Valid: userInfo.Department != ""}
	} else if err != nil {
		return nil, "", err
	}

	// Обновляем отдел
	if userInfo.Department != "" {
		_, _ = h.db.Exec(
			"UPDATE users SET department = ? WHERE id = ?",
			userInfo.Department,
			user.ID,
		)
		user.Department = sql.NullString{String: userInfo.Department, Valid: true}
	}

	departmentStr := ""
	if user.Department.Valid {
		departmentStr = user.Department.String
	}

	// Генерируем JWT
	token, err := database.GenerateJWT(user.ID, user.Username, user.Role, departmentStr)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}

func (h *AuthHandler) loginLocal(username, password string) (*models.User, string, error) {
	var user models.User

	log.Printf("loginLocal: querying for local user %s", username)

	err := h.db.QueryRow(
		"SELECT id, username, password_hash, role, department, is_ad_user FROM users WHERE username = ? AND is_ad_user = 0",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Department, &user.IsADUser)

	if err == sql.ErrNoRows {
		log.Printf("loginLocal: user %s not found in local database", username)
		return nil, "", fmt.Errorf("user not found")
	} else if err != nil {
		log.Printf("loginLocal: database error for user %s: %v", username, err)
		return nil, "", err
	}

	log.Printf("loginLocal: user %s found, checking password", username)

	// Проверяем пароль
	if !database.CheckPasswordHash(password, user.PasswordHash) {
		log.Printf("loginLocal: invalid password for user %s", username)
		return nil, "", fmt.Errorf("invalid credentials")
	}

	log.Printf("loginLocal: password valid for user %s, generating token", username)

	departmentStr := ""
	if user.Department.Valid {
		departmentStr = user.Department.String
	}

	// Генерируем JWT
	token, err := database.GenerateJWT(user.ID, user.Username, user.Role, departmentStr)
	if err != nil {
		log.Printf("loginLocal: failed to generate JWT for user %s: %v", username, err)
		return nil, "", err
	}

	return &user, token, nil
}

func (h *AuthHandler) returnLoginResponse(c *gin.Context, user *models.User, token string) {
	department := ""
	if user.Department.Valid {
		department = user.Department.String
	}

	responseUser := gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"department": department,
		"is_ad_user": user.IsADUser,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Авторизация успешна",
		"token":   token,
		"user":    responseUser,
	})
}

// GetDepartments возвращает список доступных отделов
func (h *AuthHandler) GetDepartments(c *gin.Context) {
	departments := []gin.H{
		{
			"value":    "",
			"label":    "Выбор отдела",
			"disabled": true,
		},
		{
			"value":    "ОП",
			"label":    "ОП",
			"disabled": false,
		},
		{
			"value":    "ОВ",
			"label":    "ОВ",
			"disabled": false,
		},
		{
			"value":    "РП",
			"label":    "РП",
			"disabled": false,
		},
		{
			"value":    "ГИП",
			"label":    "ГИП",
			"disabled": false,
		},
		{
			"value":    "ПС",
			"label":    "ПС",
			"disabled": false,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"departments": departments,
	})
}

// GetProfile возвращает профиль текущего пользователя
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, ok := userIDInterface.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User

	err := h.db.QueryRow(`
		SELECT id, username, role, department, is_ad_user, created_at
		FROM users
		WHERE id = ?
	`, userID).Scan(&user.ID, &user.Username, &user.Role, &user.Department, &user.IsADUser, &user.CreatedAt)

	if err != nil {
		log.Printf("GetProfile error: failed to get user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении профиля"})
		return
	}

	department := ""
	if user.Department.Valid {
		department = user.Department.String
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"department": department,
		"is_ad_user": user.IsADUser,
		"created_at": user.CreatedAt,
	})
}

// UpdateDepartment обновляет отдел пользователя
func (h *AuthHandler) UpdateDepartment(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		log.Printf("UpdateDepartment error: userID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, ok := userIDInterface.(int)
	if !ok {
		log.Printf("UpdateDepartment error: userID is not int, type: %T", userIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	log.Printf("UpdateDepartment: received request, userID from context: %d", userID)

	var req struct {
		Department string `json:"department" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("UpdateDepartment error - JSON bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("UpdateDepartment: updating department for user %d to %s", userID, req.Department)

	result, err := h.db.Exec(
		"UPDATE users SET department = ?, updated_at = ? WHERE id = ?",
		req.Department, time.Now(), userID,
	)

	if err != nil {
		log.Printf("UpdateDepartment error: failed to update department for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении отдела"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("UpdateDepartment: successfully updated user %d, rows affected: %d", userID, rowsAffected)

	// Получаем обновленные данные пользователя
	var user models.User
	err = h.db.QueryRow(
		"SELECT id, username, role, department, is_ad_user FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Role, &user.Department, &user.IsADUser)

	if err != nil {
		log.Printf("UpdateDepartment error: failed to get updated user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении обновленных данных"})
		return
	}

	// Генерируем новый JWT с обновленным отделом
	departmentStr := ""
	if user.Department.Valid {
		departmentStr = user.Department.String
	}

	newToken, err := database.GenerateJWT(user.ID, user.Username, user.Role, departmentStr)
	if err != nil {
		log.Printf("UpdateDepartment error: failed to generate new JWT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при генерации токена"})
		return
	}

	log.Printf("UpdateDepartment: generated new JWT token for user %d with department %s", userID, departmentStr)

	c.JSON(http.StatusOK, gin.H{
		"message": "Отдел успешно обновлен",
		"token":   newToken,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"department": departmentStr,
			"is_ad_user": user.IsADUser,
		},
	})
}

// SetDepartmentOnFirstLogin устанавливает отдел для нового AD пользователя при первом входе
// Это публичный эндпоинт (не требует авторизации)
func (h *AuthHandler) SetDepartmentOnFirstLogin(c *gin.Context) {
	var req struct {
		Token      string `json:"token" binding:"required"`
		Department string `json:"department" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("SetDepartmentOnFirstLogin error - JSON bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("SetDepartmentOnFirstLogin: received request with token")

	// Проверяем и парсим token
	claims := &database.Claims{}
	token, err := jwt.ParseWithClaims(req.Token, claims, func(token *jwt.Token) (interface{}, error) {
		return database.JwtKey, nil
	})

	if err != nil || !token.Valid {
		log.Printf("SetDepartmentOnFirstLogin error: invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	userID := claims.UserID
	log.Printf("SetDepartmentOnFirstLogin: extracted userID from token: %d", userID)

	// Обновляем отдел в БД
	result, err := h.db.Exec(
		"UPDATE users SET department = ?, updated_at = ? WHERE id = ?",
		req.Department, time.Now(), userID,
	)

	if err != nil {
		log.Printf("SetDepartmentOnFirstLogin error: failed to update department for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении отдела"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("SetDepartmentOnFirstLogin: successfully updated user %d, rows affected: %d", userID, rowsAffected)

	// Получаем обновленные данные пользователя
	var user models.User
	err = h.db.QueryRow(
		"SELECT id, username, role, department, is_ad_user FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Role, &user.Department, &user.IsADUser)

	if err != nil {
		log.Printf("SetDepartmentOnFirstLogin error: failed to get updated user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении обновленных данных"})
		return
	}

	// Генерируем новый JWT с обновленным отделом
	departmentStr := ""
	if user.Department.Valid {
		departmentStr = user.Department.String
	}

	newToken, err := database.GenerateJWT(user.ID, user.Username, user.Role, departmentStr)
	if err != nil {
		log.Printf("SetDepartmentOnFirstLogin error: failed to generate new JWT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при генерации токена"})
		return
	}

	log.Printf("SetDepartmentOnFirstLogin: generated new JWT token for user %d with department %s", userID, departmentStr)

	c.JSON(http.StatusOK, gin.H{
		"message": "Отдел успешно установлен",
		"token":   newToken,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"department": departmentStr,
			"is_ad_user": user.IsADUser,
		},
	})
}
