package models

import "time"

type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Course struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Level       string    `json:"level"`
	Tags        []string  `json:"tags"`
	AuthorID    int       `json:"author_id"`
	IsPublic    bool      `json:"is_public"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Lesson struct {
	ID              int    `json:"id"`
	CourseID        int    `json:"course_id"`
	Title           string `json:"title"`
	Type            string `json:"type"`
	Order           int    `json:"order"`
	DurationMinutes int    `json:"duration_minutes"`
}

type UserProgress struct {
	UserID   int  `json:"user_id"`
	LessonID int  `json:"lesson_id"`
	Done     bool `json:"done"`
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     Role   `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
