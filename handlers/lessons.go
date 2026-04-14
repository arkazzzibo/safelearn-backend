package handlers

import (
	"net/http"
	"safelearn-backend/db"
	"safelearn-backend/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetLessons(c *gin.Context) {
	courseID, err := strconv.Atoi(c.Param("course_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID курса"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, course_id, title, type, ord, duration_minutes
		FROM lessons WHERE course_id = $1 ORDER BY ord
	`, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения уроков"})
		return
	}
	defer rows.Close()

	var lessons []models.Lesson
	for rows.Next() {
		var l models.Lesson
		rows.Scan(&l.ID, &l.CourseID, &l.Title, &l.Type, &l.Order, &l.DurationMinutes)
		lessons = append(lessons, l)
	}
	if lessons == nil {
		lessons = []models.Lesson{}
	}
	c.JSON(http.StatusOK, lessons)
}

func CreateLesson(c *gin.Context) {
	courseID, err := strconv.Atoi(c.Param("course_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID курса"})
		return
	}

	var lesson models.Lesson
	if err := c.ShouldBindJSON(&lesson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lesson.CourseID = courseID

	err = db.DB.QueryRow(`
		INSERT INTO lessons (course_id, title, type, ord, duration_minutes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, lesson.CourseID, lesson.Title, lesson.Type, lesson.Order, lesson.DurationMinutes,
	).Scan(&lesson.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания урока"})
		return
	}
	c.JSON(http.StatusCreated, lesson)
}

func UpdateLesson(c *gin.Context) {
	lessonID, err := strconv.Atoi(c.Param("lesson_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID урока"})
		return
	}

	var lesson models.Lesson
	if err := c.ShouldBindJSON(&lesson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE lessons SET title=$1, type=$2, ord=$3, duration_minutes=$4
		WHERE id=$5
	`, lesson.Title, lesson.Type, lesson.Order, lesson.DurationMinutes, lessonID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления урока"})
		return
	}

	lesson.ID = lessonID
	c.JSON(http.StatusOK, lesson)
}

func DeleteLesson(c *gin.Context) {
	lessonID, err := strconv.Atoi(c.Param("lesson_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID урока"})
		return
	}

	_, err = db.DB.Exec("DELETE FROM lessons WHERE id=$1", lessonID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления урока"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Урок удалён"})
}

func GetBlocks(c *gin.Context) {
	lessonID, err := strconv.Atoi(c.Param("lesson_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID урока"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, lesson_id, type, ord, is_ib, content::text
		FROM lesson_blocks WHERE lesson_id=$1 ORDER BY ord
	`, lessonID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения блоков"})
		return
	}
	defer rows.Close()

	var blocks []map[string]interface{}
	for rows.Next() {
		var id, lessonId, ord int
		var blockType string
		var isIb bool
		var content string
		rows.Scan(&id, &lessonId, &blockType, &ord, &isIb, &content)
		blocks = append(blocks, map[string]interface{}{
			"id": id, "lesson_id": lessonId, "type": blockType,
			"ord": ord, "is_ib": isIb, "content": content,
		})
	}
	if blocks == nil {
		blocks = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, blocks)
}

func SaveBlock(c *gin.Context) {
	lessonID, err := strconv.Atoi(c.Param("lesson_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID урока"})
		return
	}

	var body struct {
		ID      int    `json:"id"`
		Type    string `json:"type"`
		Ord     int    `json:"ord"`
		IsIb    bool   `json:"is_ib"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Content == "" {
		body.Content = "{}"
	}

	var blockID int
	if body.ID == 0 {
		err = db.DB.QueryRow(`
			INSERT INTO lesson_blocks (lesson_id, type, ord, is_ib, content)
			VALUES ($1, $2, $3, $4, $5::jsonb)
			RETURNING id
		`, lessonID, body.Type, body.Ord, body.IsIb, body.Content).Scan(&blockID)
	} else {
		_, err = db.DB.Exec(`
			UPDATE lesson_blocks SET type=$1, ord=$2, is_ib=$3, content=$4::jsonb
			WHERE id=$5
		`, body.Type, body.Ord, body.IsIb, body.Content, body.ID)
		blockID = body.ID
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения блока"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": blockID, "message": "Блок сохранён"})
}

func DeleteBlock(c *gin.Context) {
	blockID, err := strconv.Atoi(c.Param("block_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID блока"})
		return
	}
	_, err = db.DB.Exec("DELETE FROM lesson_blocks WHERE id=$1", blockID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления блока"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Блок удалён"})
}
