package ldap

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"time"

	"task-management-backend/crypto"
	"task-management-backend/models"

	ldap "github.com/go-ldap/ldap/v3"
)

type LDAPManager struct {
	db *sql.DB
}

type LDAPUserInfo struct {
	Username   string
	Email      string
	Department string
	FirstName  string
	LastName   string
}

func NewLDAPManager(db *sql.DB) *LDAPManager {
	return &LDAPManager{db: db}
}

// GetADConfig получает конфигурацию AD из БД
func (lm *LDAPManager) GetADConfig() (*models.ADConfig, error) {
	var config models.ADConfig
	var encryptedPassword string
	err := lm.db.QueryRow(`
		SELECT id, enabled, directory_type, server_url, base_dn, bind_dn, 
		       bind_password, user_search_base, user_name_attr, department_attr, 
		       email_attr, group_search_base, sync_interval, tls_enabled, 
		       certificate_path, skip_cert_verify, created_at, updated_at
		FROM ad_config
		ORDER BY id DESC
		LIMIT 1
	`).Scan(
		&config.ID, &config.Enabled, &config.DirectoryType, &config.ServerURL,
		&config.BaseDN, &config.BindDN, &encryptedPassword, &config.UserSearchBase,
		&config.UserNameAttr, &config.DepartmentAttr, &config.EmailAttr,
		&config.GroupSearchBase, &config.SyncInterval, &config.TLSEnabled,
		&config.CertificatePath, &config.SkipCertVerify, &config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Расшифровываем пароль
	if encryptedPassword != "" {
		encryptor, err := crypto.LoadKeyFromEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to load encryption key: %w", err)
		}

		decrypted, err := encryptor.Decrypt(encryptedPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}

		config.BindPassword = decrypted
	}

	return &config, nil
}

// SaveADConfig сохраняет конфигурацию AD в БД
func (lm *LDAPManager) SaveADConfig(config *models.ADConfig) error {
	// Шифруем пароль
	encryptor, err := crypto.LoadKeyFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load encryption key: %w", err)
	}

	encryptedPassword, err := encryptor.Encrypt(config.BindPassword)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	result, err := lm.db.Exec(`
		INSERT INTO ad_config (
			enabled, directory_type, server_url, base_dn, bind_dn, bind_password,
			user_search_base, user_name_attr, department_attr, email_attr,
			group_search_base, sync_interval, tls_enabled, certificate_path, 
			skip_cert_verify, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		config.Enabled, config.DirectoryType, config.ServerURL, config.BaseDN,
		config.BindDN, encryptedPassword, config.UserSearchBase, config.UserNameAttr,
		config.DepartmentAttr, config.EmailAttr, config.GroupSearchBase,
		config.SyncInterval, config.TLSEnabled, config.CertificatePath,
		config.SkipCertVerify, time.Now(), time.Now(),
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	config.ID = int(id)
	return nil
}

// UpdateADConfig обновляет конфигурацию AD
func (lm *LDAPManager) UpdateADConfig(config *models.ADConfig) error {
	// Шифруем пароль
	encryptor, err := crypto.LoadKeyFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load encryption key: %w", err)
	}

	encryptedPassword, err := encryptor.Encrypt(config.BindPassword)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	_, err = lm.db.Exec(`
		UPDATE ad_config
		SET enabled = ?, directory_type = ?, server_url = ?, base_dn = ?,
		    bind_dn = ?, bind_password = ?, user_search_base = ?,
		    user_name_attr = ?, department_attr = ?, email_attr = ?,
		    group_search_base = ?, sync_interval = ?, tls_enabled = ?,
		    certificate_path = ?, skip_cert_verify = ?, updated_at = ?
		WHERE id = ?
	`,
		config.Enabled, config.DirectoryType, config.ServerURL, config.BaseDN,
		config.BindDN, encryptedPassword, config.UserSearchBase,
		config.UserNameAttr, config.DepartmentAttr, config.EmailAttr,
		config.GroupSearchBase, config.SyncInterval, config.TLSEnabled,
		config.CertificatePath, config.SkipCertVerify, time.Now(), config.ID,
	)

	return err
}

// dialLDAP создаёт подключение к LDAP серверу с поддержкой TLS
func (lm *LDAPManager) dialLDAP(config *models.ADConfig) (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if config.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.SkipCertVerify,
		}

		// Если указан сертификат и не пропускаем проверку - загружаем его
		if config.CertificatePath != "" && !config.SkipCertVerify {
			caCert, err := tls.LoadX509KeyPair(config.CertificatePath, config.CertificatePath)
			if err != nil {
				log.Printf("Warning: could not load certificate from %s: %v, will use InsecureSkipVerify\n", config.CertificatePath, err)
			} else {
				tlsConfig.Certificates = []tls.Certificate{caCert}
			}
		}

		conn, err = ldap.DialURL(config.ServerURL, ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(config.ServerURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %v", err)
	}

	return conn, nil
}

// AuthenticateADUser аутентифицирует пользователя в LDAP/AD
func (lm *LDAPManager) AuthenticateADUser(username, password string) (*LDAPUserInfo, error) {
	config, err := lm.GetADConfig()
	if err != nil || config == nil || !config.Enabled {
		return nil, fmt.Errorf("AD is not configured")
	}

	// Подключаемся к LDAP серверу
	conn, err := lm.dialLDAP(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Биндимся с сервисным аккаунтом для поиска пользователя
	err = conn.Bind(config.BindDN, config.BindPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to LDAP server: %v", err)
	}

	// Ищем пользователя в LDAP
	filter := fmt.Sprintf("(%s=%s)", config.UserNameAttr, ldap.EscapeFilter(username))
	searchRequest := ldap.NewSearchRequest(
		config.UserSearchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{config.UserNameAttr, config.EmailAttr, config.DepartmentAttr, "cn", "mail"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search LDAP: %v", err)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("user not found in LDAP")
	}

	if len(sr.Entries) > 1 {
		return nil, fmt.Errorf("multiple users found in LDAP")
	}

	userEntry := sr.Entries[0]
	userDN := userEntry.DN

	// Закрываем старое соединение и подключаемся с учётными данными пользователя
	conn.Close()
	conn, err = lm.dialLDAP(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Пытаемся авторизоваться под пользователем
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %v", err)
	}

	// Извлекаем информацию о пользователе
	userInfo := &LDAPUserInfo{
		Username:   userEntry.GetAttributeValue(config.UserNameAttr),
		Email:      userEntry.GetAttributeValue(config.EmailAttr),
		Department: userEntry.GetAttributeValue(config.DepartmentAttr),
		FirstName:  userEntry.GetAttributeValue("givenName"),
		LastName:   userEntry.GetAttributeValue("sn"),
	}

	if userInfo.Username == "" {
		userInfo.Username = username
	}
	if userInfo.Email == "" {
		userInfo.Email = userEntry.GetAttributeValue("mail")
	}

	return userInfo, nil
}

// SyncADUsers синхронизирует пользователей из AD
func (lm *LDAPManager) SyncADUsers() (int, error) {
	config, err := lm.GetADConfig()
	if err != nil || config == nil || !config.Enabled {
		return 0, fmt.Errorf("AD is not configured")
	}

	conn, err := lm.dialLDAP(config)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	err = conn.Bind(config.BindDN, config.BindPassword)
	if err != nil {
		return 0, fmt.Errorf("failed to bind to LDAP server: %v", err)
	}

	// Поиск всех пользователей
	filter := "(objectClass=*)"
	searchRequest := ldap.NewSearchRequest(
		config.UserSearchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{config.UserNameAttr, config.EmailAttr, config.DepartmentAttr, "mail"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return 0, fmt.Errorf("failed to search LDAP: %v", err)
	}

	synced := 0
	for _, entry := range sr.Entries {
		username := entry.GetAttributeValue(config.UserNameAttr)
		if username == "" {
			continue
		}

		email := entry.GetAttributeValue(config.EmailAttr)
		if email == "" {
			email = entry.GetAttributeValue("mail")
		}

		department := entry.GetAttributeValue(config.DepartmentAttr)

		// Проверяем, существует ли пользователь уже
		var exists bool
		var currentDepartment sql.NullString
		err := lm.db.QueryRow(
			"SELECT COUNT(*) > 0 FROM users WHERE username = ? AND is_ad_user = 1",
			username,
		).Scan(&exists)

		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking user %s: %v\n", username, err)
			continue
		}

		if exists {
			// Получаем текущий отдел пользователя
			err := lm.db.QueryRow(
				"SELECT department FROM users WHERE username = ? AND is_ad_user = 1",
				username,
			).Scan(&currentDepartment)

			if err != nil && err != sql.ErrNoRows {
				log.Printf("Error getting current department for user %s: %v\n", username, err)
				continue
			}

			// Обновляем пользователя только если отдел в AD не пустой ИЛИ у пользователя отдел не установлен
			if department != "" || !currentDepartment.Valid || currentDepartment.String == "" {
				_, err = lm.db.Exec(`
					UPDATE users
					SET department = ?, updated_at = ?
					WHERE username = ? AND is_ad_user = 1
				`, sql.NullString{String: department, Valid: department != ""}, time.Now(), username)

				if err != nil {
					log.Printf("Error updating user %s: %v\n", username, err)
					continue
				}
			} else {
				// Отдел уже установлен пользователем, не обновляем
				log.Printf("Keeping existing department for user %s\n", username)
			}
		} else {
			// Создаём нового пользователя
			_, err = lm.db.Exec(`
				INSERT INTO users (username, password_hash, role, department, is_ad_user, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`,
				username,
				"",     // Пользователи AD не имеют локального пароля
				"user", // По умолчанию роль "user"
				sql.NullString{String: department, Valid: department != ""},
				true,
				time.Now(),
				time.Now(),
			)

			if err != nil {
				log.Printf("Error creating user %s: %v\n", username, err)
				continue
			}
		}

		synced++
	}

	return synced, nil
}
