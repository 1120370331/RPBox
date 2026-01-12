package database

import (
	"fmt"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&model.User{},
		&model.Profile{},
		&model.ProfileVersion{},
	); err != nil {
		return err
	}

	DB = db
	return nil
}
