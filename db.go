package paymentservice

import (
	"os"
	"sync"

	"github.com/go-pg/pg/v10"
)

var (
	dbOnce sync.Once
	db     *pg.DB
)

func GetDB() *pg.DB {
	dbOnce.Do(func() {
		host := firstEnv("DB_HOST", "POSTGRES_HOST")
		port := firstEnv("DB_PORT", "POSTGRES_PORT")
		db = pg.Connect(&pg.Options{
			Addr:     host + ":" + port,
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Database: os.Getenv("POSTGRES_DB"),
			PoolSize: 20,
		})
	})
	return db
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
