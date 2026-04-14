package handlers

import (
	"encoding/json"
	"net/http"
	"safelearn-backend/db"

	"github.com/gin-gonic/gin"
)

type SaveResultRequest struct {
	BlockID   int             `json:"block_id" binding:"required"`
	Score     int             `json:"score"`
	MaxScore  int             `json:"max_score"`
	Passed    bool            `json:"passed"`
	Answers   json.RawMessage `json:"answers"`
	BlockType string          `json:"block_type"`
}

func SaveResult(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req SaveResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	answers := req.Answers
	if answers == nil {
		answers = json.RawMessage("{}")
	}

	var resultID int
	err := db.DB.QueryRow(`
		INSERT INTO user_results (user_id, block_id, score, max_score, passed, answers)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		RETURNING id
	`, userID, req.BlockID, req.Score, req.MaxScore, req.Passed, string(answers)).Scan(&resultID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результата"})
		return
	}

	go UpdateCompetencies(userID.(int), req.BlockID, req.Score, req.MaxScore)

	eventType := "quiz_passed"
	if !req.Passed {
		eventType = "quiz_failed"
	}
	if req.BlockType == "phishing" {
		eventType = "phishing_simulated"
	} else if req.BlockType == "scenario" {
		eventType = "scenario_completed"
	}

	LogActivity(userID.(int), eventType, "block", req.BlockID, map[string]interface{}{
		"score": req.Score, "max_score": req.MaxScore, "passed": req.Passed,
	})

	c.JSON(http.StatusCreated, gin.H{"id": resultID, "message": "Результат сохранён"})
}

func SavePhishingAttempt(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		BlockID    int  `json:"block_id"`
		IsPhishing bool `json:"is_phishing"`
		UserAnswer bool `json:"user_answer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.DB.Exec(`
		INSERT INTO phishing_attempts (user_id, block_id, is_phishing, user_answer)
		VALUES ($1, $2, $3, $4)
	`, userID, req.BlockID, req.IsPhishing, req.UserAnswer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения"})
		return
	}

	isCorrect := req.IsPhishing == req.UserAnswer
	LogActivity(userID.(int), "phishing_simulated", "block", req.BlockID,
		map[string]interface{}{"is_correct": isCorrect})

	c.JSON(http.StatusCreated, gin.H{"correct": isCorrect})
}

func GetMyResults(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`
		SELECT r.id, r.block_id, r.score, r.max_score, r.passed,
		       to_char(r.completed_at, 'YYYY-MM-DD HH24:MI'),
		       COALESCE(lb.type, '') as block_type,
		       COALESCE(l.title, '') as lesson_title,
		       COALESCE(c.title, '') as course_title
		FROM user_results r
		LEFT JOIN lesson_blocks lb ON lb.id = r.block_id
		LEFT JOIN lessons l ON l.id = lb.lesson_id
		LEFT JOIN courses c ON c.id = l.course_id
		WHERE r.user_id = $1
		ORDER BY r.completed_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения результатов"})
		return
	}
	defer rows.Close()

	type Result struct {
		ID          int    `json:"id"`
		BlockID     int    `json:"block_id"`
		Score       int    `json:"score"`
		MaxScore    int    `json:"max_score"`
		Passed      bool   `json:"passed"`
		CompletedAt string `json:"completed_at"`
		BlockType   string `json:"block_type"`
		LessonTitle string `json:"lesson_title"`
		CourseTitle string `json:"course_title"`
	}

	var results []Result
	for rows.Next() {
		var r Result
		rows.Scan(&r.ID, &r.BlockID, &r.Score, &r.MaxScore, &r.Passed,
			&r.CompletedAt, &r.BlockType, &r.LessonTitle, &r.CourseTitle)
		results = append(results, r)
	}
	if results == nil {
		results = []Result{}
	}
	c.JSON(http.StatusOK, results)
}

func GetCompetencies(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`
		SELECT t.name, t.color, uc.score,
		       to_char(uc.updated_at, 'YYYY-MM-DD') as updated_at
		FROM user_competencies uc
		JOIN tags t ON t.id = uc.tag_id
		WHERE uc.user_id = $1
		ORDER BY uc.score DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения компетенций"})
		return
	}
	defer rows.Close()

	type Competency struct {
		TagName   string `json:"tag_name"`
		Color     string `json:"color"`
		Score     int    `json:"score"`
		UpdatedAt string `json:"updated_at"`
	}

	var competencies []Competency
	for rows.Next() {
		var comp Competency
		rows.Scan(&comp.TagName, &comp.Color, &comp.Score, &comp.UpdatedAt)
		competencies = append(competencies, comp)
	}
	if competencies == nil {
		competencies = []Competency{}
	}
	c.JSON(http.StatusOK, competencies)
}

func UpdateCompetencies(userID, blockID, score, maxScore int) {
	if maxScore == 0 {
		return
	}
	pct := int(float64(score) / float64(maxScore) * 100)

	rows, err := db.DB.Query(`
		SELECT DISTINCT t.id, t.name
		FROM lesson_blocks lb
		JOIN lessons l ON l.id = lb.lesson_id
		JOIN course_tags ct ON ct.course_id = l.course_id
		JOIN tags t ON t.id = ct.tag_id
		WHERE lb.id = $1
	`, blockID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tagID int
		var tagName string
		rows.Scan(&tagID, &tagName)

		db.DB.Exec(`
			INSERT INTO user_competencies (user_id, tag_id, score)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, tag_id) DO UPDATE
			SET score = LEAST(100, (user_competencies.score + $3) / 2),
			    updated_at = NOW()
		`, userID, tagID, pct)

		db.DB.Exec(`
			INSERT INTO competency_history (user_id, tag_id, score_delta, reason)
			VALUES ($1, $2, $3, $4)
		`, userID, tagID, pct, "Результат теста: #"+tagName)
	}
}
