package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	var req models.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Register error - JSON bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Register attempt: username=%s, department=%s, firstName=%s, lastName=%s",
		req.Username, req.Department, req.FirstName, req.LastName)

	// Проверка существования пользователя
	var exists bool
	err := h.db.QueryRow("SELECT 1 FROM users WHERE username = ?", req.Username).Scan(&exists)
	if err == nil {
		log.Printf("Register failed: user %s already exists", req.Username)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Такой пользователь уже существует"})
		return
	}

	// Валидация ФИО (первое имя и фамилия обязательны)
	if req.FirstName == "" || req.LastName == "" {
		log.Printf("Register failed: empty first_name or last_name for user %s", req.Username)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Имя и фамилия обязательны"})
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
		"INSERT INTO users (username, password_hash, role, department, first_name, last_name, patronymic, is_ad_user, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		req.Username, hashedPassword, "user", req.Department, req.FirstName, req.LastName, req.Patronymic, false, time.Now(), time.Now(),
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
			"first_name": req.FirstName,
			"last_name":  req.LastName,
			"patronymic": req.Patronymic,
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

	log.Printf("loginAD: LDAP authentication successful for %s, got info: firstName=%s, lastName=%s, dept=%s",
		username, userInfo.FirstName, userInfo.LastName, userInfo.Department)

	// Проверяем, существует ли пользователь в нашей БД
	var user models.User
	var firstName, lastName, patronymic sql.NullString

	err = h.db.QueryRow(
		"SELECT id, username, role, department, is_ad_user, first_name, last_name, patronymic FROM users WHERE username = ? AND is_ad_user = 1",
		username,
	).Scan(&user.ID, &user.Username, &user.Role, &user.Department, &user.IsADUser, &firstName, &lastName, &patronymic)

	if err == sql.ErrNoRows {
		// Пользователя нет в БД, создаём его
		log.Printf("loginAD: user %s not found in DB, creating new user", username)
		result, err := h.db.Exec(`
			INSERT INTO users (username, password_hash, role, department, first_name, last_name, patronymic, is_ad_user, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			username,
			"", // У доменных пользователей нет локального пароля
			"user",
			sql.NullString{String: userInfo.Department, Valid: userInfo.Department != ""},
			userInfo.FirstName,
			userInfo.LastName,
			"",
			true,
			time.Now(),
			time.Now(),
		)

		if err != nil {
			log.Printf("loginAD: failed to insert new user %s: %v", username, err)
			return nil, "", err
		}

		userID, _ := result.LastInsertId()
		user.ID = int(userID)
		user.Username = username
		user.Role = "user"
		user.IsADUser = true
		user.Department = sql.NullString{String: userInfo.Department, Valid: userInfo.Department != ""}
		user.FirstName = userInfo.FirstName
		user.LastName = userInfo.LastName
		user.Patronymic = ""

		log.Printf("loginAD: created new user %s with id %d, firstName=%s, lastName=%s",
			username, userID, userInfo.FirstName, userInfo.LastName)

		// Перечитываем новопо пользователя из БД, чтобы получить актуальные значения
		var dept, fname, lname, pname sql.NullString
		err = h.db.QueryRow(
			"SELECT id, username, role, department, is_ad_user, first_name, last_name, patronymic FROM users WHERE id = ?",
			user.ID,
		).Scan(&user.ID, &user.Username, &user.Role, &dept, &user.IsADUser, &fname, &lname, &pname)

		if err == nil {
			if dept.Valid {
				user.Department = dept
			}
			if fname.Valid {
				user.FirstName = fname.String
			}
			if lname.Valid {
				user.LastName = lname.String
			}
			if pname.Valid {
				user.Patronymic = pname.String
			}
		}
	} else if err != nil {
		log.Printf("loginAD: error querying user %s: %v", username, err)
		return nil, "", err
	} else {
		log.Printf("loginAD: user %s found in DB, updating from LDAP info", username)
		// Конвертируем NullString в обычные строки
		if firstName.Valid {
			user.FirstName = firstName.String
		}
		if lastName.Valid {
			user.LastName = lastName.String
		}
		if patronymic.Valid {
			user.Patronymic = patronymic.String
		}
	}

	// Обновляем отдел и ФИО из AD
	if userInfo.Department != "" || userInfo.FirstName != "" || userInfo.LastName != "" {
		log.Printf("loginAD: updating user %s with LDAP info: firstName=%s, lastName=%s, dept=%s",
			username, userInfo.FirstName, userInfo.LastName, userInfo.Department)

		// Если department пусто в LDAP, не перезаписываем существующий department в БД
		if userInfo.Department == "" {
			// Если department пусто в LDAP, обновляем только ФИО
			result, err := h.db.Exec(
				"UPDATE users SET first_name = ?, last_name = ? WHERE id = ?",
				userInfo.FirstName,
				userInfo.LastName,
				user.ID,
			)

			if err != nil {
				log.Printf("loginAD: error updating user %s: %v", username, err)
				return nil, "", err
			}

			rowsAffected, _ := result.RowsAffected()
			log.Printf("loginAD: updated %d rows for user %s (ФИО only)", rowsAffected, username)
		} else {
			// Если department заполнен в LDAP, обновляем все
			result, err := h.db.Exec(
				"UPDATE users SET department = ?, first_name = ?, last_name = ? WHERE id = ?",
				userInfo.Department,
				userInfo.FirstName,
				userInfo.LastName,
				user.ID,
			)

			if err != nil {
				log.Printf("loginAD: error updating user %s: %v", username, err)
				return nil, "", err
			}

			rowsAffected, _ := result.RowsAffected()
			log.Printf("loginAD: updated %d rows for user %s", rowsAffected, username)
		}

		// Перечитываем user из БД, чтобы получить актуальные значения
		var dept, fname, lname, pname sql.NullString
		err = h.db.QueryRow(
			"SELECT id, username, role, department, is_ad_user, first_name, last_name, patronymic FROM users WHERE id = ?",
			user.ID,
		).Scan(&user.ID, &user.Username, &user.Role, &dept, &user.IsADUser, &fname, &lname, &pname)

		if err != nil {
			log.Printf("loginAD: error re-reading user %s from DB: %v", username, err)
			// Не критично, используем то, что установили вручную
		} else {
			if dept.Valid {
				user.Department = dept
			}
			if fname.Valid {
				user.FirstName = fname.String
			}
			if lname.Valid {
				user.LastName = lname.String
			}
			if pname.Valid {
				user.Patronymic = pname.String
			}
			log.Printf("loginAD: re-read user %s from DB: dept=%s, firstName=%s, lastName=%s",
				username, user.Department.String, user.FirstName, user.LastName)
		}
	}

	departmentStr := ""
	if user.Department.Valid {
		departmentStr = user.Department.String
	}

	// Генерируем JWT
	token, err := database.GenerateJWT(user.ID, user.Username, user.Role, departmentStr)
	if err != nil {
		log.Printf("loginAD: failed to generate JWT for user %s: %v", username, err)
		return nil, "", err
	}

	log.Printf("loginAD: successfully authenticated user %s", username)
	return &user, token, nil
}

func (h *AuthHandler) loginLocal(username, password string) (*models.User, string, error) {
	var user models.User
	var firstName, lastName, patronymic sql.NullString

	log.Printf("loginLocal: querying for local user %s", username)

	err := h.db.QueryRow(
		"SELECT id, username, password_hash, role, department, is_ad_user, password_reset_required, first_name, last_name, patronymic FROM users WHERE username = ? AND is_ad_user = 0",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Department, &user.IsADUser, &user.PasswordResetRequired, &firstName, &lastName, &patronymic)

	if err == sql.ErrNoRows {
		log.Printf("loginLocal: user %s not found in local database", username)
		return nil, "", fmt.Errorf("user not found")
	} else if err != nil {
		log.Printf("loginLocal: database error for user %s: %v", username, err)
		return nil, "", err
	}

	// Конвертируем NullString в обычные строки
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if patronymic.Valid {
		user.Patronymic = patronymic.String
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

	log.Printf("returnLoginResponse: preparing response for user %s with firstName=%s, lastName=%s, patronymic=%s",
		user.Username, user.FirstName, user.LastName, user.Patronymic)

	responseUser := gin.H{
		"id":                      user.ID,
		"username":                user.Username,
		"role":                    user.Role,
		"department":              department,
		"is_ad_user":              user.IsADUser,
		"password_reset_required": user.PasswordResetRequired,
		"first_name":              user.FirstName,
		"last_name":               user.LastName,
		"patronymic":              user.Patronymic,
	}

	log.Printf("returnLoginResponse: sending response user object: %+v", responseUser)

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
	var firstName, lastName, patronymic sql.NullString

	err := h.db.QueryRow(`
		SELECT id, username, role, department, is_ad_user, password_reset_required, first_name, last_name, patronymic, created_at
		FROM users
		WHERE id = ?
	`, userID).Scan(&user.ID, &user.Username, &user.Role, &user.Department, &user.IsADUser, &user.PasswordResetRequired, &firstName, &lastName, &patronymic, &user.CreatedAt)

	if err != nil {
		log.Printf("GetProfile error: failed to get user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении профиля"})
		return
	}

	// Конвертируем NullString в обычные строки
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if patronymic.Valid {
		user.Patronymic = patronymic.String
	}

	department := ""
	if user.Department.Valid {
		department = user.Department.String
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                      user.ID,
		"username":                user.Username,
		"role":                    user.Role,
		"department":              department,
		"is_ad_user":              user.IsADUser,
		"password_reset_required": user.PasswordResetRequired,
		"first_name":              user.FirstName,
		"last_name":               user.LastName,
		"patronymic":              user.Patronymic,
		"created_at":              user.CreatedAt,
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

// ResetPassword сбрасывает пароль пользователя (только для админов)
// Устанавливает флаг password_reset_required в true
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currentUserID := c.GetInt("userID")
	var user models.User

	// Получаем информацию о пользователе
	err = h.db.QueryRow(
		"SELECT id, username, role, is_ad_user FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Role, &user.IsADUser)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	} else if err != nil {
		log.Printf("ResetPassword error: database query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении информации о пользователе"})
		return
	}

	// Нельзя сбросить пароль AD пользователям
	if user.IsADUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя сбросить пароль пользователю из Active Directory"})
		return
	}

	log.Printf("ResetPassword: admin %d is resetting password for user %d (%s)", currentUserID, userID, user.Username)

	// Устанавливаем флаг password_reset_required
	result, err := h.db.Exec(
		"UPDATE users SET password_reset_required = 1, updated_at = ? WHERE id = ?",
		time.Now(), userID,
	)

	if err != nil {
		log.Printf("ResetPassword error: failed to update user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сбросе пароля"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	log.Printf("ResetPassword: password reset flag set for user %d", userID)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Пароль пользователя %s отмечен к сбросу. Пользователю необходимо установить новый пароль при входе.", user.Username),
	})
}

// SetNewPasswordAfterReset позволяет пользователю установить новый пароль после сброса
func (h *AuthHandler) SetNewPasswordAfterReset(c *gin.Context) {
	userID := c.GetInt("userID")

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("SetNewPasswordAfterReset error - JSON bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("SetNewPasswordAfterReset: user %d is setting new password", userID)

	var user models.User

	// Проверяем, что флаг password_reset_required установлен
	err := h.db.QueryRow(
		"SELECT id, username, password_reset_required FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.PasswordResetRequired)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	} else if err != nil {
		log.Printf("SetNewPasswordAfterReset error: database query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке данных"})
		return
	}

	if !user.PasswordResetRequired {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Сброс пароля не требуется"})
		return
	}

	// Хешируем новый пароль
	hashedPassword, err := database.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("SetNewPasswordAfterReset error: failed to hash password for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обработке пароля"})
		return
	}

	// Обновляем пароль и сбрасываем флаг
	result, err := h.db.Exec(
		"UPDATE users SET password_hash = ?, password_reset_required = 0, updated_at = ? WHERE id = ?",
		hashedPassword, time.Now(), userID,
	)

	if err != nil {
		log.Printf("SetNewPasswordAfterReset error: failed to update user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при установке нового пароля"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	log.Printf("SetNewPasswordAfterReset: password successfully updated for user %d", userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Пароль успешно установлен",
	})
}

// ChangePassword позволяет пользователю изменить пароль самостоятельно
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetInt("userID")

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ChangePassword error - JSON bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("ChangePassword: user %d is changing password", userID)

	var user models.User

	// Получаем текущий хеш пароля
	err := h.db.QueryRow(
		"SELECT id, username, password_hash, is_ad_user FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsADUser)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	} else if err != nil {
		log.Printf("ChangePassword error: database query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении данных"})
		return
	}

	// Пользователи AD не могут менять пароль локально
	if user.IsADUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пользователи Active Directory не могут менять пароль в системе"})
		return
	}

	// Проверяем текущий пароль
	if !database.CheckPasswordHash(req.CurrentPassword, user.PasswordHash) {
		log.Printf("ChangePassword: invalid current password for user %d", userID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Текущий пароль неверен"})
		return
	}

	// Хешируем новый пароль
	hashedPassword, err := database.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("ChangePassword error: failed to hash password for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обработке пароля"})
		return
	}

	// Обновляем пароль
	result, err := h.db.Exec(
		"UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?",
		hashedPassword, time.Now(), userID,
	)

	if err != nil {
		log.Printf("ChangePassword error: failed to update user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при изменении пароля"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	log.Printf("ChangePassword: password successfully changed for user %d", userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Пароль успешно изменен",
	})
}

// Logout инвалидирует текущий токен пользователя
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetInt("userID")
	token, exists := c.Get("token")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token not found in context"})
		return
	}

	tokenString, ok := token.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token format"})
		return
	}

	// Декодируем токен чтобы получить время истечения
	claims := &database.Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return database.JwtKey, nil
	})

	if err != nil {
		log.Printf("Logout: failed to parse token for user %d: %v", userID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	// Добавляем токен в чёрный список
	expiresAt := claims.ExpiresAt.Time
	err = database.AddTokenToBlacklist(h.db, userID, tokenString, expiresAt)
	if err != nil {
		log.Printf("Logout: failed to add token to blacklist for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	log.Printf("Logout: user %d successfully logged out, token added to blacklist", userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully logged out",
	})
}
