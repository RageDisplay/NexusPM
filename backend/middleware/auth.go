package middleware

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"task-management-backend/database"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func AuthMiddleware() gin.HandlerFunc {
	return AuthMiddlewareWithDB(nil)
}

func AuthMiddlewareWithDB(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// Проверяем чёрный список, если БД доступна
		if db != nil && database.IsTokenBlacklisted(db, tokenString) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
			c.Abort()
			return
		}

		claims := &database.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return database.JwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("userDepartment", claims.Department)
		c.Set("token", tokenString)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func ManagerOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		if role != "manager" && role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Manager or admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminOrSelf - разрешить админу видеть всё, или пользователю видеть только свои данные
func AdminOrSelf() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		userID, _ := c.Get("userID")

		// Админ может делать что угодно
		if role == "admin" {
			c.Next()
			return
		}

		// Обычный пользователь может видеть только свои данные
		requestedID := c.Param("id")
		if requestedID != "" {
			userIDStr := strconv.Itoa(userID.(int))
			if requestedID != userIDStr {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
