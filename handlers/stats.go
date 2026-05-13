package handlers

import (
	"net/http"
	"safelearn-backend/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetCourseStats(c *gin.Context) {
	courseID, err := strconv.Atoi(c.Param("course_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID курса"})
		return
	}

	// Проверяем что курс принадлежит преподавателю
	authorID, _ := c.Get("user_id")
	var ownerID int
	err = db.DB.QueryRow(
		"SELECT author_id FROM courses WHERE id=$1", courseID,
	).Scan(&ownerID)
	if err != nil || ownerID != authorID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Нет доступа к этому курсу"})
		return
	}

	// Общие данные курса
	var totalLessons, totalEnrolled, totalCompleted int
	db.DB.QueryRow(
		"SELECT COUNT(*) FROM lessons WHERE course_id=$1", courseID,
	).Scan(&totalLessons)

	db.DB.QueryRow(
		"SELECT COUNT(*) FROM course_enrollments WHERE course_id=$1", courseID,
	).Scan(&totalEnrolled)

	db.DB.QueryRow(
		"SELECT COUNT(*) FROM course_enrollments WHERE course_id=$1 AND completed_at IS NOT NULL", courseID,
	).Scan(&totalCompleted)

	// Список студентов — считаем уникальные пройденные уроки через подзапрос
	rows, err := db.DB.Query(`
		SELECT
			u.id,
			u.name,
			u.email,
			to_char(ce.enrolled_at, 'DD.MM.YYYY'),
			COALESCE(to_char(ce.completed_at, 'DD.MM.YYYY'), ''),
			(
				SELECT COUNT(DISTINCT up.lesson_id)
				FROM user_progress up
				JOIN lessons l ON l.id = up.lesson_id
				WHERE up.user_id = u.id
				  AND l.course_id = $1
				  AND up.done = true
			) as lessons_done,
			(
				SELECT COUNT(r.id)
				FROM user_results r
				JOIN lesson_blocks lb ON lb.id = r.block_id
				JOIN lessons l ON l.id = lb.lesson_id
				WHERE r.user_id = u.id AND l.course_id = $1
			) as tests_total,
			(
				SELECT COUNT(r.id)
				FROM user_results r
				JOIN lesson_blocks lb ON lb.id = r.block_id
				JOIN lessons l ON l.id = lb.lesson_id
				WHERE r.user_id = u.id AND l.course_id = $1 AND r.passed = true
			) as tests_passed,
			COALESCE((
				SELECT AVG(r.score::float / NULLIF(r.max_score, 0) * 100)
				FROM user_results r
				JOIN lesson_blocks lb ON lb.id = r.block_id
				JOIN lessons l ON l.id = lb.lesson_id
				WHERE r.user_id = u.id AND l.course_id = $1
			), 0) as avg_score
		FROM course_enrollments ce
		JOIN users u ON u.id = ce.user_id
		WHERE ce.course_id = $1
		ORDER BY lessons_done DESC, u.name
	`, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения студентов: " + err.Error()})
		return
	}
	defer rows.Close()

	type StudentStat struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		Email        string  `json:"email"`
		EnrolledAt   string  `json:"enrolled_at"`
		CompletedAt  string  `json:"completed_at"`
		LessonsDone  int     `json:"lessons_done"`
		TotalLessons int     `json:"total_lessons"`
		ProgressPct  int     `json:"progress_pct"`
		TestsTotal   int     `json:"tests_total"`
		TestsPassed  int     `json:"tests_passed"`
		AvgScore     float64 `json:"avg_score"`
	}

	var students []StudentStat
	var totalProgress int

	for rows.Next() {
		var s StudentStat
		var avgScore float64
		rows.Scan(
			&s.ID, &s.Name, &s.Email,
			&s.EnrolledAt, &s.CompletedAt,
			&s.LessonsDone, &s.TestsTotal, &s.TestsPassed, &avgScore,
		)
		s.AvgScore = avgScore
		s.TotalLessons = totalLessons
		if totalLessons > 0 {
			s.ProgressPct = int(float64(s.LessonsDone) / float64(totalLessons) * 100)
		}
		totalProgress += s.ProgressPct
		students = append(students, s)
	}

	if students == nil {
		students = []StudentStat{}
	}

	// Средний прогресс по всем студентам
	avgProgress := 0
	if len(students) > 0 {
		avgProgress = totalProgress / len(students)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_lessons":   totalLessons,
		"total_enrolled":  totalEnrolled,
		"total_completed": totalCompleted,
		"avg_progress":    avgProgress,
		"students":        students,
	})
}
