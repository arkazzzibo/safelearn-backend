package db

import (
	"database/sql"
	"fmt"
	"log"
	"safelearn-backend/config"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect(cfg *config.Config) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}

	log.Println("✅ Подключение к PostgreSQL установлено")
	createTables()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id         SERIAL PRIMARY KEY,
			name       VARCHAR(255) NOT NULL,
			email      VARCHAR(255) UNIQUE NOT NULL,
			password   VARCHAR(255) NOT NULL,
			role       VARCHAR(50) DEFAULT 'student',
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS courses (
			id          SERIAL PRIMARY KEY,
			title       VARCHAR(255) NOT NULL,
			description TEXT,
			icon        VARCHAR(100) DEFAULT 'mdi-book',
			level       VARCHAR(50) DEFAULT 'basic',
			tags        TEXT[],
			author_id   INT REFERENCES users(id),
			is_public   BOOLEAN DEFAULT true,
			status      VARCHAR(50) DEFAULT 'draft',
			created_at  TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS lessons (
			id        SERIAL PRIMARY KEY,
			course_id INT REFERENCES courses(id) ON DELETE CASCADE,
			title     VARCHAR(255) NOT NULL,
			type      VARCHAR(50) DEFAULT 'text',
			ord       INT DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS user_progress (
			user_id   INT REFERENCES users(id) ON DELETE CASCADE,
			lesson_id INT REFERENCES lessons(id) ON DELETE CASCADE,
			done      BOOLEAN DEFAULT false,
			PRIMARY KEY (user_id, lesson_id)
		)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatalf("Ошибка создания таблицы: %v", err)
		}
	}

	log.Println("✅ Таблицы готовы")
}
