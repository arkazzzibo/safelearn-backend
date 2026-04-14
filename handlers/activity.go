package handlers

import (
	"encoding/json"
	"net/http"
	"safelearn-backend/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetActivity — лента активности пользователя
func GetActivity(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`
		SELECT id, type,
		       COALESCE(entity_type, '') as entity_type,
		       COALESCE(entity_id, 0) as entity_id,
		       COALESCE(metadata::text, '{}') as metadata,
		       to_char(created_at, 'DD Mon, HH24:MI') as created_at
		FROM activity_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения активности"})
		return
	}
	defer rows.Close()

	type ActivityItem struct {
		ID         int    `json:"id"`
		Type       string `json:"type"`
		EntityType string `json:"entity_type"`
		EntityID   int    `json:"entity_id"`
		Metadata   string `json:"metadata"`
		CreatedAt  string `json:"created_at"`
	}

	var items []ActivityItem
	for rows.Next() {
		var item ActivityItem
		rows.Scan(&item.ID, &item.Type, &item.EntityType,
			&item.EntityID, &item.Metadata, &item.CreatedAt)
		items = append(items, item)
	}
	if items == nil {
		items = []ActivityItem{}
	}

	c.JSON(http.StatusOK, items)
}

// SearchCourses — поиск курсов с фильтрами
func SearchCourses(c *gin.Context) {
	query := c.Query("q")
	level := c.Query("level")
	tag := c.Query("tag")

	sqlStr := `
		SELECT DISTINCT c.id, c.title, c.description, c.icon, c.level,
		       c.author_id, c.is_public, c.status,
		       to_char(c.created_at, 'YYYY-MM-DD') as created_at
		FROM courses c
		LEFT JOIN course_tags ct ON ct.course_id = c.id
		LEFT JOIN tags t ON t.id = ct.tag_id
		WHERE c.is_public = true AND c.status = 'published'
	`
	args := []interface{}{}
	argN := 1

	if query != "" {
		sqlStr += " AND (LOWER(c.title) LIKE LOWER($" + strconv.Itoa(argN) + ") OR LOWER(c.description) LIKE LOWER($" + strconv.Itoa(argN) + "))"
		args = append(args, "%"+query+"%")
		argN++
	}
	if level != "" {
		sqlStr += " AND c.level = $" + strconv.Itoa(argN)
		args = append(args, level)
		argN++
	}
	if tag != "" {
		sqlStr += " AND t.name = $" + strconv.Itoa(argN)
		args = append(args, tag)
		argN++
	}

	sqlStr += " ORDER BY c.created_at DESC LIMIT 20"

	rows, err := db.DB.Query(sqlStr, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка поиска"})
		return
	}
	defer rows.Close()

	type CourseResult struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Level       string `json:"level"`
		AuthorID    int    `json:"author_id"`
		IsPublic    bool   `json:"is_public"`
		Status      string `json:"status"`
		CreatedAt   string `json:"created_at"`
	}

	var courses []CourseResult
	for rows.Next() {
		var course CourseResult
		rows.Scan(&course.ID, &course.Title, &course.Description,
			&course.Icon, &course.Level, &course.AuthorID,
			&course.IsPublic, &course.Status, &course.CreatedAt)
		courses = append(courses, course)
	}
	if courses == nil {
		courses = []CourseResult{}
	}

	c.JSON(http.StatusOK, courses)
}

// LogActivity — записать событие в лог (экспортируемая функция)
func LogActivity(userID int, actType, entityType string, entityID int, meta interface{}) {
	metaJSON, _ := json.Marshal(meta)
	db.DB.Exec(`
		INSERT INTO activity_log (user_id, type, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, userID, actType, entityType, entityID, string(metaJSON))
}
