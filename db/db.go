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
	var dsn string

	if cfg.DatabaseURL != "" {
		// Railway даёт готовый URL — используем напрямую
		dsn = cfg.DatabaseURL
		log.Println("🔗 Используем DATABASE_URL (Railway)")
	} else {
		// Локальная разработка
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
		)
		log.Println("🔗 Используем локальную БД")
	}

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
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS organizations (
			id         SERIAL PRIMARY KEY,
			name       VARCHAR(255) NOT NULL,
			domain     VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS categories (
			id          SERIAL PRIMARY KEY,
			name        VARCHAR(255) NOT NULL,
			slug        VARCHAR(255) UNIQUE NOT NULL,
			icon        VARCHAR(100),
			description TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS tags (
			id    SERIAL PRIMARY KEY,
			name  VARCHAR(100) UNIQUE NOT NULL,
			color VARCHAR(7) DEFAULT '#378ADD'
		)`,

		`CREATE TABLE IF NOT EXISTS courses (
			id              SERIAL PRIMARY KEY,
			title           VARCHAR(255) NOT NULL,
			description     TEXT,
			icon            VARCHAR(100) DEFAULT 'mdi-book',
			level           VARCHAR(50) DEFAULT 'basic',
			status          VARCHAR(50) DEFAULT 'draft',
			is_public       BOOLEAN DEFAULT true,
			author_id       INT REFERENCES users(id),
			organization_id INT REFERENCES organizations(id),
			created_at      TIMESTAMP DEFAULT NOW(),
			updated_at      TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS course_categories (
			course_id   INT REFERENCES courses(id) ON DELETE CASCADE,
			category_id INT REFERENCES categories(id) ON DELETE CASCADE,
			PRIMARY KEY (course_id, category_id)
		)`,

		`CREATE TABLE IF NOT EXISTS course_tags (
			course_id INT REFERENCES courses(id) ON DELETE CASCADE,
			tag_id    INT REFERENCES tags(id) ON DELETE CASCADE,
			PRIMARY KEY (course_id, tag_id)
		)`,

		`CREATE TABLE IF NOT EXISTS lessons (
			id               SERIAL PRIMARY KEY,
			course_id        INT REFERENCES courses(id) ON DELETE CASCADE,
			title            VARCHAR(255) NOT NULL,
			type             VARCHAR(50) DEFAULT 'text',
			ord              INT DEFAULT 0,
			duration_minutes INT DEFAULT 0,
			created_at       TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS lesson_blocks (
			id         SERIAL PRIMARY KEY,
			lesson_id  INT REFERENCES lessons(id) ON DELETE CASCADE,
			type       VARCHAR(50) NOT NULL,
			ord        INT DEFAULT 0,
			is_ib      BOOLEAN DEFAULT false,
			content    JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS course_enrollments (
			id           SERIAL PRIMARY KEY,
			user_id      INT REFERENCES users(id) ON DELETE CASCADE,
			course_id    INT REFERENCES courses(id) ON DELETE CASCADE,
			enrolled_at  TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP,
			UNIQUE (user_id, course_id)
		)`,

		`CREATE TABLE IF NOT EXISTS user_progress (
			user_id    INT REFERENCES users(id) ON DELETE CASCADE,
			lesson_id  INT REFERENCES lessons(id) ON DELETE CASCADE,
			done       BOOLEAN DEFAULT false,
			started_at TIMESTAMP DEFAULT NOW(),
			done_at    TIMESTAMP,
			time_spent INT DEFAULT 0,
			PRIMARY KEY (user_id, lesson_id)
		)`,

		`CREATE TABLE IF NOT EXISTS user_results (
			id           SERIAL PRIMARY KEY,
			user_id      INT REFERENCES users(id) ON DELETE CASCADE,
			block_id     INT REFERENCES lesson_blocks(id) ON DELETE CASCADE,
			score        INT DEFAULT 0,
			max_score    INT DEFAULT 0,
			passed       BOOLEAN DEFAULT false,
			answers      JSONB,
			attempts     INT DEFAULT 1,
			completed_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS phishing_attempts (
			id          SERIAL PRIMARY KEY,
			user_id     INT REFERENCES users(id) ON DELETE CASCADE,
			block_id    INT REFERENCES lesson_blocks(id) ON DELETE CASCADE,
			is_phishing BOOLEAN NOT NULL,
			user_answer BOOLEAN NOT NULL,
			is_correct  BOOLEAN GENERATED ALWAYS AS (is_phishing = user_answer) STORED,
			answered_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS user_competencies (
			user_id    INT REFERENCES users(id) ON DELETE CASCADE,
			tag_id     INT REFERENCES tags(id) ON DELETE CASCADE,
			score      INT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (user_id, tag_id)
		)`,

		`CREATE TABLE IF NOT EXISTS competency_history (
			id          SERIAL PRIMARY KEY,
			user_id     INT REFERENCES users(id) ON DELETE CASCADE,
			tag_id      INT REFERENCES tags(id) ON DELETE CASCADE,
			score_delta INT NOT NULL,
			reason      VARCHAR(255),
			recorded_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS activity_log (
			id          SERIAL PRIMARY KEY,
			user_id     INT REFERENCES users(id) ON DELETE CASCADE,
			type        VARCHAR(100) NOT NULL,
			entity_type VARCHAR(50),
			entity_id   INT,
			metadata    JSONB,
			created_at  TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS notifications (
			id         SERIAL PRIMARY KEY,
			user_id    INT REFERENCES users(id) ON DELETE CASCADE,
			type       VARCHAR(100) NOT NULL,
			title      VARCHAR(255) NOT NULL,
			body       TEXT,
			is_read    BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_lessons_course_id ON lessons(course_id)`,
		`CREATE INDEX IF NOT EXISTS idx_lesson_blocks_lesson_id ON lesson_blocks(lesson_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_progress_user_id ON user_progress(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_course_enrollments_user_id ON course_enrollments(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_log_user_id ON activity_log(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id)`,
	}
	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Printf("⚠️  %v", err)
		}
	}

	log.Println("✅ Таблицы готовы")
}
