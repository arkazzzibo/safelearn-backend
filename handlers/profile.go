package handlers

import (
	"net/http"
	"safelearn-backend/db"

	"github.com/gin-gonic/gin"
)

type CompetencyItem struct {
	TagName string `json:"tag_name"`
	Color   string `json:"color"`
	Score   int    `json:"score"`
}

type ActivityItem struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	EntityType string `json:"entity_type"`
	EntityID   int    `json:"entity_id"`
	Metadata   string `json:"metadata"`
	CreatedAt  string `json:"created_at"`
}

type CourseProgress struct {
	CourseID    int    `json:"course_id"`
	CourseTitle string `json:"course_title"`
	Icon        string `json:"icon"`
	Total       int    `json:"total_lessons"`
	Done        int    `json:"done_lessons"`
	Percent     int    `json:"percent"`
}

type ProfileStats struct {
	CoursesEnrolled  int `json:"courses_enrolled"`
	CoursesCompleted int `json:"courses_completed"`
	LessonsDone      int `json:"lessons_done"`
	TestsPassed      int `json:"tests_passed"`
}

type ProfileResponse struct {
	Stats        ProfileStats     `json:"stats"`
	Competencies []CompetencyItem `json:"competencies"`
	Activity     []ActivityItem   `json:"activity"`
	Courses      []CourseProgress `json:"courses"`
}

// GetProfile — полные данные профиля для текущего пользователя
func GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var resp ProfileResponse

	// Статистика
	db.DB.QueryRow(`
		SELECT COUNT(*) FROM course_enrollments WHERE user_id = $1
	`, userID).Scan(&resp.Stats.CoursesEnrolled)

	db.DB.QueryRow(`
		SELECT COUNT(*) FROM course_enrollments
		WHERE user_id = $1 AND completed_at IS NOT NULL
	`, userID).Scan(&resp.Stats.CoursesCompleted)

	db.DB.QueryRow(`
		SELECT COUNT(*) FROM user_progress
		WHERE user_id = $1 AND done = true
	`, userID).Scan(&resp.Stats.LessonsDone)

	db.DB.QueryRow(`
		SELECT COUNT(*) FROM user_results
		WHERE user_id = $1 AND passed = true
	`, userID).Scan(&resp.Stats.TestsPassed)

	// Компетенции — из user_competencies + tags
	rows, err := db.DB.Query(`
		SELECT t.name, t.color, uc.score
		FROM user_competencies uc
		JOIN tags t ON t.id = uc.tag_id
		WHERE uc.user_id = $1
		ORDER BY uc.score DESC
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item CompetencyItem
			rows.Scan(&item.TagName, &item.Color, &item.Score)
			resp.Competencies = append(resp.Competencies, item)
		}
	}
	if resp.Competencies == nil {
		resp.Competencies = []CompetencyItem{}
	}

	// Активность — последние 10 событий
	actRows, err := db.DB.Query(`
		SELECT id, type, COALESCE(entity_type,''), COALESCE(entity_id,0),
		       COALESCE(metadata::text,'{}'), to_char(created_at, 'DD Mon, HH24:MI')
		FROM activity_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 10
	`, userID)
	if err == nil {
		defer actRows.Close()
		for actRows.Next() {
			var item ActivityItem
			actRows.Scan(&item.ID, &item.Type, &item.EntityType,
				&item.EntityID, &item.Metadata, &item.CreatedAt)
			resp.Activity = append(resp.Activity, item)
		}
	}
	if resp.Activity == nil {
		resp.Activity = []ActivityItem{}
	}

	// Прогресс по каждому курсу студента
	cpRows, err := db.DB.Query(`
		SELECT
			c.id,
			c.title,
			c.icon,
			COUNT(l.id) AS total,
			COUNT(up.lesson_id) FILTER (WHERE up.done = true) AS done
		FROM course_enrollments ce
		JOIN courses c ON c.id = ce.course_id
		LEFT JOIN lessons l ON l.course_id = c.id
		LEFT JOIN user_progress up ON up.lesson_id = l.id AND up.user_id = ce.user_id
		WHERE ce.user_id = $1
		GROUP BY c.id, c.title, c.icon
		ORDER BY c.id
	`, userID)
	if err == nil {
		defer cpRows.Close()
		for cpRows.Next() {
			var cp CourseProgress
			cpRows.Scan(&cp.CourseID, &cp.CourseTitle, &cp.Icon, &cp.Total, &cp.Done)
			if cp.Total > 0 {
				cp.Percent = int(float64(cp.Done) / float64(cp.Total) * 100)
			}
			resp.Courses = append(resp.Courses, cp)
		}
	}
	if resp.Courses == nil {
		resp.Courses = []CourseProgress{}
	}

	c.JSON(http.StatusOK, resp)
}
