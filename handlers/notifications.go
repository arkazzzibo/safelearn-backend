package handlers

import (
	"database/sql"
	"net/http"
	"safelearn-backend/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetNotifications — список уведомлений с кол-вом непрочитанных
func GetNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`
		SELECT id, type, title, COALESCE(body,''), is_read,
		       to_char(created_at, 'DD Mon, HH24:MI')
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения уведомлений"})
		return
	}
	defer rows.Close()

	type Notification struct {
		ID        int    `json:"id"`
		Type      string `json:"type"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		IsRead    bool   `json:"is_read"`
		CreatedAt string `json:"created_at"`
	}

	var notifications []Notification
	for rows.Next() {
		var n Notification
		rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.IsRead, &n.CreatedAt)
		notifications = append(notifications, n)
	}
	if notifications == nil {
		notifications = []Notification{}
	}

	var unread int
	db.DB.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND is_read=false",
		userID,
	).Scan(&unread)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"unread_count":  unread,
	})
}

// MarkNotificationRead — отметить одно уведомление прочитанным
func MarkNotificationRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notifID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	_, err = db.DB.Exec(
		"UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2",
		notifID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Прочитано"})
}

// MarkAllNotificationsRead — отметить все прочитанными
func MarkAllNotificationsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db.DB.Exec("UPDATE notifications SET is_read=true WHERE user_id=$1", userID)
	c.JSON(http.StatusOK, gin.H{"message": "Все прочитаны"})
}

// CreateNotification — создать уведомление (внутренняя функция)
func CreateNotification(userID int, notifType, title, body string) {
	db.DB.Exec(`
		INSERT INTO notifications (user_id, type, title, body)
		VALUES ($1, $2, $3, $4)
	`, userID, notifType, title, body)
}

// вспомогательная — используется внутри пакета
var _ = sql.ErrNoRows
