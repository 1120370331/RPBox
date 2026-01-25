package testutil

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewTestDB creates an isolated in-memory SQLite database and migrates models.
func NewTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()

	name := sanitizeTestName(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
	}

	return db
}

func sanitizeTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
	cleaned := replacer.Replace(name)
	if cleaned == "" {
		return "testdb"
	}
	return cleaned
}
