package handlers

import (
	"net/http"
	"safelearn-backend/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

// EnrollCourse — записаться на курс
func EnrollCourse(c *gin.Context) {
	userID, _ := c.Get("user_id")
	courseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID курса"})
		return
	}

	var courseTitle string
	err = db.DB.QueryRow("SELECT title FROM courses WHERE id=$1", courseID).Scan(&courseTitle)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Курс не найден"})
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO course_enrollments (user_id, course_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, course_id) DO NOTHING
	`, userID, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка записи на курс"})
		return
	}

	LogActivity(userID.(int), "course_enrolled", "course", courseID, map[string]string{
		"course_title": courseTitle,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Записан на курс", "course_id": courseID})
}

// IsEnrolled — проверить запись на курс
func IsEnrolled(c *gin.Context) {
	userID, _ := c.Get("user_id")
	courseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	var enrolled bool
	db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM course_enrollments WHERE user_id=$1 AND course_id=$2
		)
	`, userID, courseID).Scan(&enrolled)

	c.JSON(http.StatusOK, gin.H{"enrolled": enrolled})
}

// GetEnrolledCourses — все курсы студента
func GetEnrolledCourses(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`
		SELECT c.id, c.title, c.description, c.icon, c.level, c.status,
		       ce.enrolled_at, ce.completed_at
		FROM course_enrollments ce
		JOIN courses c ON c.id = ce.course_id
		WHERE ce.user_id = $1
		ORDER BY ce.enrolled_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения курсов"})
		return
	}
	defer rows.Close()

	type EnrolledCourse struct {
		ID          int     `json:"id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Icon        string  `json:"icon"`
		Level       string  `json:"level"`
		Status      string  `json:"status"`
		EnrolledAt  string  `json:"enrolled_at"`
		CompletedAt *string `json:"completed_at"`
	}

	var courses []EnrolledCourse
	for rows.Next() {
		var course EnrolledCourse
		rows.Scan(
			&course.ID, &course.Title, &course.Description,
			&course.Icon, &course.Level, &course.Status,
			&course.EnrolledAt, &course.CompletedAt,
		)
		courses = append(courses, course)
	}
	if courses == nil {
		courses = []EnrolledCourse{}
	}

	c.JSON(http.StatusOK, courses)
}
