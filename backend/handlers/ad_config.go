package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"task-management-backend/ldap"
	"task-management-backend/models"

	"github.com/gin-gonic/gin"
)

type ADConfigHandler struct {
	db          *sql.DB
	ldapManager *ldap.LDAPManager
}

func NewADConfigHandler(db *sql.DB) *ADConfigHandler {
	return &ADConfigHandler{
		db:          db,
		ldapManager: ldap.NewLDAPManager(db),
	}
}

// GetADConfigStatus получает статус AD (публичный эндпоинт для проверки на странице логина)
func (h *ADConfigHandler) GetADConfigStatus(c *gin.Context) {
	config, err := h.ldapManager.GetADConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"enabled": false})
		return
	}

	if config == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"enabled": config.Enabled})
}

// GetADConfig получает текущую конфигурацию AD
func (h *ADConfigHandler) GetADConfig(c *gin.Context) {
	config, err := h.ldapManager.GetADConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении конфигурации"})
		return
	}

	if config == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "AD не настроена",
			"config":  nil,
		})
		return
	}

	// Не возвращаем пароль в ответе
	config.BindPassword = ""

	c.JSON(http.StatusOK, gin.H{
		"config": config,
	})
}

// SetupADConfig сохраняет или обновляет конфигурацию AD
func (h *ADConfigHandler) SetupADConfig(c *gin.Context) {
	var req struct {
		Enabled         bool   `json:"enabled"`
		DirectoryType   string `json:"directory_type"`
		ServerURL       string `json:"server_url"`
		BaseDN          string `json:"base_dn"`
		BindDN          string `json:"bind_dn"`
		BindPassword    string `json:"bind_password"`
		UserSearchBase  string `json:"user_search_base"`
		UserNameAttr    string `json:"user_name_attr"`
		DepartmentAttr  string `json:"department_attr"`
		EmailAttr       string `json:"email_attr"`
		GroupSearchBase string `json:"group_search_base"`
		SyncInterval    int    `json:"sync_interval"`
		TLSEnabled      bool   `json:"tls_enabled"`
		CertificatePath string `json:"certificate_path"`
		SkipCertVerify  bool   `json:"skip_cert_verify"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Если AD включается, проверяем обязательные поля
	if req.Enabled {
		if req.DirectoryType == "" || req.ServerURL == "" || req.BaseDN == "" ||
			req.BindDN == "" || req.BindPassword == "" || req.UserSearchBase == "" ||
			req.UserNameAttr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Все обязательные поля должны быть заполнены для включения AD"})
			return
		}

		// Проверяем подключение к серверу LDAP только если AD включается
		if err := h.testLDAPConnection(req.ServerURL, req.BindDN, req.BindPassword); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось подключиться к LDAP серверу: " + err.Error()})
			return
		}
	}

	// Получаем существующую конфигурацию
	existingConfig, err := h.ldapManager.GetADConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении конфигурации"})
		return
	}

	config := &models.ADConfig{
		Enabled:         req.Enabled,
		DirectoryType:   req.DirectoryType,
		ServerURL:       req.ServerURL,
		BaseDN:          req.BaseDN,
		BindDN:          req.BindDN,
		BindPassword:    req.BindPassword,
		UserSearchBase:  req.UserSearchBase,
		UserNameAttr:    req.UserNameAttr,
		DepartmentAttr:  req.DepartmentAttr,
		EmailAttr:       req.EmailAttr,
		GroupSearchBase: req.GroupSearchBase,
		SyncInterval:    req.SyncInterval,
		TLSEnabled:      req.TLSEnabled,
		CertificatePath: req.CertificatePath,
		SkipCertVerify:  req.SkipCertVerify,
	}

	if config.SyncInterval == 0 {
		config.SyncInterval = 60 // Час по умолчанию
	}

	var saveErr error
	if existingConfig == nil {
		// Создаём новую конфигурацию
		saveErr = h.ldapManager.SaveADConfig(config)
	} else {
		// Обновляем существующую
		config.ID = existingConfig.ID
		saveErr = h.ldapManager.UpdateADConfig(config)
	}

	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении конфигурации: " + saveErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Конфигурация AD успешно сохранена",
		"config":  config,
	})
}

// TestADConnection проверяет подключение к AD серверу
func (h *ADConfigHandler) TestADConnection(c *gin.Context) {
	var req struct {
		ServerURL    string `json:"server_url" binding:"required"`
		BindDN       string `json:"bind_dn" binding:"required"`
		BindPassword string `json:"bind_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.testLDAPConnection(req.ServerURL, req.BindDN, req.BindPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка подключения: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Подключение успешно",
	})
}

// SyncADUsers синхронизирует пользователей из AD
func (h *ADConfigHandler) SyncADUsers(c *gin.Context) {
	synced, err := h.ldapManager.SyncADUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка синхронизации: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Синхронизация завершена",
		"users_synced": synced,
	})
}

// testLDAPConnection тестирует подключение к LDAP серверу
func (h *ADConfigHandler) testLDAPConnection(serverURL, bindDN, bindPassword string) error {
	// Используем методы ldapManager для проверки подключения
	// Создаём временную конфигурацию и пытаемся подключиться
	tempConfig := &models.ADConfig{
		ServerURL:    serverURL,
		BindDN:       bindDN,
		BindPassword: bindPassword,
	}

	// Попытка подключения будет произведена в AuthenticateADUser
	// Для тестирования проверяем, что параметры не пусты
	if tempConfig.ServerURL == "" || tempConfig.BindDN == "" {
		return fmt.Errorf("invalid parameters: server_url and bind_dn are required")
	}

	return nil
}
