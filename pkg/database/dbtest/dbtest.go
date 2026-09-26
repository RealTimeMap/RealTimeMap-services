// Package dbtest открывает локальный Postgres для интеграционных тестов.
//
// Тест, которому нужна настоящая БД, пропускается, если её нет: такие тесты
// проверяют SQL, который нельзя проверить моками, но не должны ронять
// go test ./... на машине без Postgres.
//
// Параметры по умолчанию совпадают с config/config.yaml сервисов и
// переопределяются через TEST_DB_HOST, TEST_DB_PORT, TEST_DB_USER,
// TEST_DB_PASSWORD.
package dbtest

import (
	"os"
	"strconv"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"gorm.io/gorm"
)

// Open подключается к базе dbName и закрывает подключение по окончании теста.
func Open(t *testing.T, dbName string) *gorm.DB {
	t.Helper()

	port, err := strconv.Atoi(env("TEST_DB_PORT", "5432"))
	if err != nil {
		t.Fatalf("TEST_DB_PORT: %v", err)
	}

	db, err := database.New(database.Config{
		// 127.0.0.1, а не localhost: на Windows localhost сначала резолвится
		// в ::1, где Postgres из докера обычно не слушает.
		Host:     env("TEST_DB_HOST", "127.0.0.1"),
		Port:     port,
		User:     env("TEST_DB_USER", "postgres"),
		Password: env("TEST_DB_PASSWORD", "admin"),
		DBName:   dbName,
		SSLMode:  "disable",
	}, nil)
	if err != nil {
		t.Skipf("database %s not available: %v", dbName, err)
	}

	t.Cleanup(func() { _ = database.Close(db) })
	return db
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
