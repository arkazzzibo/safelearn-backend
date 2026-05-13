package handlers

import (
	"net/http"
	"safelearn-backend/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CompleteLesson — отметить урок пройденным
func CompleteLesson(c *gin.Context) {
	userID, _ := c.Get("user_id")
	lessonID, err := strconv.Atoi(c.Param("lesson_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID урока"})
		return
	}

	// Получаем данные урока и курса
	var lessonTitle, courseTitle string
	var courseID int
	db.DB.QueryRow(`
		SELECT l.title, c.title, c.id
		FROM lessons l
		JOIN courses c ON c.id = l.course_id
		WHERE l.id = $1
	`, lessonID).Scan(&lessonTitle, &courseTitle, &courseID)

	// Сохраняем прогресс
	_, err = db.DB.Exec(`
		INSERT INTO user_progress (user_id, lesson_id, done, done_at)
		VALUES ($1, $2, true, NOW())
		ON CONFLICT (user_id, lesson_id) DO UPDATE
		SET done = true, done_at = NOW()
	`, userID, lessonID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения прогресса"})
		return
	}

	// Логируем активность
	LogActivity(userID.(int), "lesson_completed", "lesson", lessonID, map[string]string{
		"lesson_title": lessonTitle,
		"course_title": courseTitle,
	})

	// Проверяем завершение курса
	// total — все уроки курса
	var total int
	db.DB.QueryRow(
		"SELECT COUNT(*) FROM lessons WHERE course_id=$1", courseID,
	).Scan(&total)

	// done — уроки курса которые студент прошёл (через JOIN чтобы не считать уроки других курсов)
	var done int
	db.DB.QueryRow(`
		SELECT COUNT(DISTINCT up.lesson_id)
		FROM user_progress up
		JOIN lessons l ON l.id = up.lesson_id
		WHERE up.user_id = $1
		  AND l.course_id = $2
		  AND up.done = true
	`, userID, courseID).Scan(&done)

	courseDone := total > 0 && done >= total

	if courseDone {
		// Проставляем completed_at только если ещё не проставлено
		db.DB.Exec(`
			UPDATE course_enrollments
			SET completed_at = NOW()
			WHERE user_id = $1
			  AND course_id = $2
			  AND completed_at IS NULL
		`, userID, courseID)

		LogActivity(userID.(int), "course_completed", "course", courseID, map[string]string{
			"course_title": courseTitle,
		})

		CreateNotification(userID.(int), "course_completed",
			"Курс завершён: "+courseTitle,
			"Поздравляем! Ты прошёл курс полностью.")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Урок пройден",
		"course_done": courseDone,
		"done":        done,
		"total":       total,
	})
}

// GetProgress — прогресс студента по курсу
func GetProgress(c *gin.Context) {
	userID, _ := c.Get("user_id")
	courseID, err := strconv.Atoi(c.Param("course_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID курса"})
		return
	}

	// Возвращаем прогресс по всем урокам курса (включая непройденные — с done=false)
	rows, err := db.DB.Query(`
		SELECT
			l.id as lesson_id,
			COALESCE(up.done, false) as done,
			COALESCE(up.time_spent, 0) as time_spent,
			COALESCE(to_char(up.done_at, 'YYYY-MM-DD HH24:MI'), '') as done_at
		FROM lessons l
		LEFT JOIN user_progress up
			ON up.lesson_id = l.id AND up.user_id = $1
		WHERE l.course_id = $2
		ORDER BY l.ord
	`, userID, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения прогресса"})
		return
	}
	defer rows.Close()

	type LessonProgress struct {
		LessonID  int    `json:"lesson_id"`
		Done      bool   `json:"done"`
		TimeSpent int    `json:"time_spent"`
		DoneAt    string `json:"done_at"`
	}

	var progress []LessonProgress
	for rows.Next() {
		var p LessonProgress
		rows.Scan(&p.LessonID, &p.Done, &p.TimeSpent, &p.DoneAt)
		progress = append(progress, p)
	}
	if progress == nil {
		progress = []LessonProgress{}
	}

	c.JSON(http.StatusOK, progress)
}
