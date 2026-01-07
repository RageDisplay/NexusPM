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
	projectHandler := handlers.NewProjectHandler(db)
	adConfigHandler := handlers.NewADConfigHandler(db)

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
	router.POST("/api/auth/set-department", authHandler.SetDepartmentOnFirstLogin)
	router.GET("/api/ad-config/status", adConfigHandler.GetADConfigStatus)
	router.GET("/api/auth/departments", authHandler.GetDepartments)

	// Защищенные маршруты
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// Авторизация
		api.GET("/auth/profile", authHandler.GetProfile)
		api.PUT("/auth/department", authHandler.UpdateDepartment)

		// Задачи
		api.GET("/tasks", taskHandler.GetTasks)
		api.POST("/tasks", taskHandler.CreateTask)
		api.PUT("/tasks/:id", taskHandler.UpdateTask)
		api.DELETE("/tasks/:id", taskHandler.DeleteTask)

		// Проекты
		api.GET("/projects", projectHandler.GetProjects)
		api.GET("/projects/user", projectHandler.GetUserProjects)
		api.GET("/projects/:id", projectHandler.GetProjectByID)
		api.POST("/projects", middleware.ManagerOrAdmin(), projectHandler.CreateProject)
		api.PUT("/projects/:id", middleware.ManagerOrAdmin(), projectHandler.UpdateProject)
		api.DELETE("/projects/:id", middleware.ManagerOrAdmin(), projectHandler.DeleteProject)
		api.POST("/projects/assign", middleware.ManagerOrAdmin(), projectHandler.AssignProjectToUser)
		api.POST("/projects/remove", middleware.ManagerOrAdmin(), projectHandler.RemoveProjectFromUser)
		api.GET("/projects/:id/users", middleware.ManagerOrAdmin(), projectHandler.GetProjectUsers)

		// Пользователи (только для админов)
		api.GET("/users", userHandler.GetUsers)
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

		// AD конфигурация (только для админов)
		api.GET("/ad-config", middleware.AdminOnly(), adConfigHandler.GetADConfig)
		api.POST("/ad-config", middleware.AdminOnly(), adConfigHandler.SetupADConfig)
		api.POST("/ad-config/test", middleware.AdminOnly(), adConfigHandler.TestADConnection)
		api.POST("/ad-config/sync", middleware.AdminOnly(), adConfigHandler.SyncADUsers)
	}

	log.Println("Server starting on :8080")
	log.Fatal(router.Run(":8080"))
}
