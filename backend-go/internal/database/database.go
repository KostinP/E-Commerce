package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDatabase() error {
	// Просто читаем переменные, без fallback!
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	// Проверяем, что все переменные установлены
	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return fmt.Errorf("missing required database environment variables: DB_HOST=%s, DB_PORT=%s, DB_USER=%s, DB_PASSWORD=%s, DB_NAME=%s",
			host, port, user, password, dbname)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	log.Println("✅ Database connected successfully")
	return nil
}

func CloseDatabase() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func GetDB() *sql.DB {
	return DB
}

func RunMigrations(db *sql.DB) error {
	mm := NewMigrationManager(db)
	return mm.Up(context.Background())
}
