package database

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var JwtKey = make([]byte, 64)

type Claims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	Department string `json:"department"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateJWT(userID int, username, role, department string) (string, error) {
	expirationTime := time.Now().Add(3 * time.Hour)
	claims := &Claims{
		UserID:     userID,
		Username:   username,
		Role:       role,
		Department: department,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}

// HashToken создаёт хэш токена для хранения в БД
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

// AddTokenToBlacklist добавляет токен в чёрный список
func AddTokenToBlacklist(db *sql.DB, userID int, token string, expiresAt time.Time) error {
	tokenHash := HashToken(token)
	_, err := db.Exec(`
		INSERT INTO token_blacklist (user_id, token_hash, expires_at)
		VALUES (?, ?, ?)
	`, userID, tokenHash, expiresAt)
	return err
}

// IsTokenBlacklisted проверяет, находится ли токен в чёрном списке
func IsTokenBlacklisted(db *sql.DB, token string) bool {
	tokenHash := HashToken(token)
	var exists bool
	err := db.QueryRow(`
		SELECT 1 FROM token_blacklist
		WHERE token_hash = ? AND expires_at > datetime('now')
	`, tokenHash).Scan(&exists)
	return err == nil && exists
}

// CleanExpiredTokens удаляет истёкшие токены из чёрного списка (можно запускать периодически)
func CleanExpiredTokens(db *sql.DB) error {
	_, err := db.Exec(`
		DELETE FROM token_blacklist
		WHERE expires_at <= datetime('now')
	`)
	return err
}
