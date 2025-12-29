package main

import (
	"log"
	"task-management-backend/database"
	"task-management-backend/handlers"
	"task-management-backend/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// Инициализация базы данных
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создание обработчиков
	authHandler := handlers.NewAuthHandler(db)
	taskHandler := handlers.NewTaskHandler(db)
	userHandler := handlers.NewUserHandler(db)
	reportHandler := handlers.NewReportHandler(db)

	router := gin.Default()

	// Разрешаем все хосты и методы (не забыть потом настроить)
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Content-Disposition")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Публичные маршруты
	router.POST("/api/register", authHandler.Register)
	router.POST("/api/login", authHandler.Login)

	// Защищенные маршруты
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// Задачи
		api.GET("/tasks", taskHandler.GetTasks)
		api.POST("/tasks", taskHandler.CreateTask)
		api.PUT("/tasks/:id", taskHandler.UpdateTask)
		api.DELETE("/tasks/:id", taskHandler.DeleteTask)

		// Пользователи (только для админов)
		api.GET("/users", middleware.AdminOnly(), userHandler.GetUsers)
		api.PUT("/users/:id/role", middleware.AdminOnly(), userHandler.UpdateUserRole)
		api.PUT("/users/:id/department", middleware.AdminOnly(), userHandler.UpdateUserDepartment)
		api.DELETE("/users/:id", middleware.AdminOnly(), userHandler.DeleteUser)

		// Отчеты
		api.GET("/reports/my-tasks", reportHandler.ExportMyTasks)
		api.GET("/reports/department-tasks", middleware.ManagerOrAdmin(), reportHandler.ExportDepartmentTasks)
		api.GET("/reports/all-tasks", middleware.AdminOnly(), reportHandler.ExportAllTasks)

		// Бэкап БД
		api.GET("/backup", middleware.AdminOnly(), handlers.BackupDB)
		api.POST("/restore", middleware.AdminOnly(), handlers.RestoreDB)
	}

	log.Println("Server starting on :8080")
	log.Fatal(router.Run(":8080"))
}
