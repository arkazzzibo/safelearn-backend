package handlers

import (
	"net/http"
	"safelearn-backend/db"
	"safelearn-backend/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

func GetCourses(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT c.id, c.title, COALESCE(c.description,''), COALESCE(c.icon,'mdi-book'),
		       COALESCE(c.level,'basic'), c.author_id, c.is_public, c.status, c.created_at,
		       COALESCE(array_agg(t.name) FILTER (WHERE t.name IS NOT NULL), '{}') as tags
		FROM courses c
		LEFT JOIN course_tags ct ON ct.course_id = c.id
		LEFT JOIN tags t ON t.id = ct.tag_id
		WHERE c.is_public = true AND c.status = 'published'
		GROUP BY c.id
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения курсов: " + err.Error()})
		return
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		var tags pq.StringArray
		err := rows.Scan(
			&course.ID, &course.Title, &course.Description,
			&course.Icon, &course.Level,
			&course.AuthorID, &course.IsPublic, &course.Status, &course.CreatedAt,
			&tags,
		)
		if err != nil {
			continue
		}
		course.Tags = []string(tags)
		if course.Tags == nil {
			course.Tags = []string{}
		}
		courses = append(courses, course)
	}
	if courses == nil {
		courses = []models.Course{}
	}
	c.JSON(http.StatusOK, courses)
}

func GetCourse(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	var course models.Course
	var tags pq.StringArray
	err = db.DB.QueryRow(`
		SELECT c.id, c.title, COALESCE(c.description,''), COALESCE(c.icon,'mdi-book'),
		       COALESCE(c.level,'basic'), c.author_id, c.is_public, c.status, c.created_at,
		       COALESCE(array_agg(t.name) FILTER (WHERE t.name IS NOT NULL), '{}') as tags
		FROM courses c
		LEFT JOIN course_tags ct ON ct.course_id = c.id
		LEFT JOIN tags t ON t.id = ct.tag_id
		WHERE c.id = $1
		GROUP BY c.id
	`, id).Scan(
		&course.ID, &course.Title, &course.Description,
		&course.Icon, &course.Level,
		&course.AuthorID, &course.IsPublic, &course.Status, &course.CreatedAt,
		&tags,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Курс не найден"})
		return
	}
	course.Tags = []string(tags)
	if course.Tags == nil {
		course.Tags = []string{}
	}

	rows, _ := db.DB.Query(
		"SELECT id, course_id, title, type, ord FROM lessons WHERE course_id=$1 ORDER BY ord",
		id,
	)
	defer rows.Close()

	var lessons []models.Lesson
	for rows.Next() {
		var l models.Lesson
		rows.Scan(&l.ID, &l.CourseID, &l.Title, &l.Type, &l.Order)
		lessons = append(lessons, l)
	}
	if lessons == nil {
		lessons = []models.Lesson{}
	}
	c.JSON(http.StatusOK, gin.H{"course": course, "lessons": lessons})
}

func CreateCourse(c *gin.Context) {
	var course models.Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authorID, _ := c.Get("user_id")
	course.AuthorID = authorID.(int)
	if course.Status == "" {
		course.Status = "draft"
	}
	if course.Icon == "" {
		course.Icon = "mdi-book"
	}

	err := db.DB.QueryRow(`
		INSERT INTO courses (title, description, icon, level, author_id, is_public, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, course.Title, course.Description, course.Icon, course.Level,
		course.AuthorID, course.IsPublic, course.Status,
	).Scan(&course.ID, &course.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания курса: " + err.Error()})
		return
	}

	saveTags(course.ID, course.Tags)
	c.JSON(http.StatusCreated, course)
}

func GetMyCourses(c *gin.Context) {
	authorID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`
		SELECT c.id, c.title, COALESCE(c.description,''), COALESCE(c.icon,'mdi-book'),
		       COALESCE(c.level,'basic'), c.author_id, c.is_public, c.status, c.created_at,
		       COALESCE(array_agg(t.name) FILTER (WHERE t.name IS NOT NULL), '{}') as tags
		FROM courses c
		LEFT JOIN course_tags ct ON ct.course_id = c.id
		LEFT JOIN tags t ON t.id = ct.tag_id
		WHERE c.author_id = $1
		GROUP BY c.id
		ORDER BY c.created_at DESC
	`, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения курсов"})
		return
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		var tags pq.StringArray
		rows.Scan(
			&course.ID, &course.Title, &course.Description,
			&course.Icon, &course.Level,
			&course.AuthorID, &course.IsPublic, &course.Status, &course.CreatedAt,
			&tags,
		)
		course.Tags = []string(tags)
		if course.Tags == nil {
			course.Tags = []string{}
		}
		courses = append(courses, course)
	}
	if courses == nil {
		courses = []models.Course{}
	}
	c.JSON(http.StatusOK, courses)
}

func UpdateCourse(c *gin.Context) {
	courseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID курса"})
		return
	}

	var course models.Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE courses
		SET title=$1, description=$2, icon=$3, level=$4, is_public=$5, status=$6
		WHERE id=$7
	`, course.Title, course.Description, course.Icon, course.Level,
		course.IsPublic, course.Status, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления курса"})
		return
	}

	if course.Tags != nil {
		db.DB.Exec("DELETE FROM course_tags WHERE course_id=$1", courseID)
		saveTags(courseID, course.Tags)
	}

	course.ID = courseID
	c.JSON(http.StatusOK, course)
}

// saveTags — сохранить теги курса через course_tags
func saveTags(courseID int, tags []string) {
	for _, tagName := range tags {
		if tagName == "" {
			continue
		}
		var tagID int
		db.DB.QueryRow(`
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name
			RETURNING id
		`, tagName).Scan(&tagID)
		if tagID > 0 {
			db.DB.Exec(`
				INSERT INTO course_tags (course_id, tag_id)
				VALUES ($1, $2) ON CONFLICT DO NOTHING
			`, courseID, tagID)
		}
	}
}
