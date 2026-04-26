package postgresql

import (
	"os"
	"sync"

	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

var (
	db   *pg.DB
	once sync.Once
)

func GetDB() *pg.DB {
	once.Do(func() {
		db = pg.Connect(&pg.Options{
			Addr:     firstEnv("DB_HOST", "POSTGRES_HOST") + ":" + firstEnv("DB_PORT", "POSTGRES_PORT"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Database: os.Getenv("POSTGRES_DB"),
			PoolSize: 20,
		})
	})
	return db
}

func InitPostgresql() *e.ErrorInfo {
	db := GetDB()

	requiredModels := []interface{}{
		(*models.Telegramuser)(nil),
		(*models.Payment)(nil),
		(*models.UserLevels)(nil),
		(*models.Mirror)(nil),
	}

	for _, model := range requiredModels {
		if err := db.Model(model).CreateTable(&orm.CreateTableOptions{IfNotExists: true}); err != nil {
			return e.FromError(err, "error creating table")
		}
	}

	return e.Nil()
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
