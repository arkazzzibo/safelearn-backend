package main

import (
	"log"
	"os"
	"safelearn-backend/config"
	"safelearn-backend/db"
	"safelearn-backend/handlers"
	"safelearn-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env не найден, используем переменные окружения")
	}

	cfg := config.Load()
	db.Connect(cfg)

	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Healthcheck
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "SafeLearn API ✅", "version": "2.0"})
	})

	// ── Авторизация (публичные) ──────────────────────────────────────
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
	}

	// ── Курсы (публичные) ────────────────────────────────────────────
	r.GET("/api/courses", handlers.GetCourses)
	r.GET("/api/courses/:id", handlers.GetCourse)
	r.GET("/api/courses/:id/lessons", handlers.GetLessons)
	r.GET("/api/lessons/:lesson_id/blocks", handlers.GetBlocks)
	r.GET("/api/search", handlers.SearchCourses)

	// ── Защищённые маршруты ──────────────────────────────────────────
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	{
		// Профиль
		api.GET("/me", handlers.Me)
		api.GET("/profile", handlers.GetProfile)

		// Запись на курс
		api.POST("/courses/:id/enroll", handlers.EnrollCourse)
		api.GET("/courses/:id/enrolled", handlers.IsEnrolled)
		api.GET("/enrolled", handlers.GetEnrolledCourses)

		// Прогресс
		api.POST("/lessons/:lesson_id/complete", handlers.CompleteLesson)
		api.GET("/progress/:course_id", handlers.GetProgress)

		// Результаты тестов
		api.POST("/results", handlers.SaveResult)
		api.GET("/results", handlers.GetMyResults)
		api.POST("/phishing-attempts", handlers.SavePhishingAttempt)

		// Компетенции
		api.GET("/competencies", handlers.GetCompetencies)

		// Активность
		api.GET("/activity", handlers.GetActivity)

		// Уведомления
		api.GET("/notifications", handlers.GetNotifications)
		api.PUT("/notifications/:id/read", handlers.MarkNotificationRead)
		api.PUT("/notifications/read-all", handlers.MarkAllNotificationsRead)

		// ── Преподаватель ────────────────────────────────────────────
		teacher := api.Group("/teacher")
		teacher.Use(middleware.TeacherRequired())
		{
			// Курсы
			teacher.GET("/courses", handlers.GetMyCourses)
			teacher.POST("/courses", handlers.CreateCourse)
			teacher.PUT("/courses/:id", handlers.UpdateCourse)

			// Уроки
			teacher.POST("/courses/:course_id/lessons", handlers.CreateLesson)
			teacher.PUT("/lessons/:lesson_id", handlers.UpdateLesson)
			teacher.DELETE("/lessons/:lesson_id", handlers.DeleteLesson)

			// Блоки
			teacher.POST("/lessons/:lesson_id/blocks", handlers.SaveBlock)
			teacher.DELETE("/blocks/:block_id", handlers.DeleteBlock)
		}
	}

	port := cfg.Port
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("🚀 SafeLearn API v2.0 запущен на порту %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Ошибка запуска: %v", err)
	}
}
