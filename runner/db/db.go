package db

import (
	"fmt"

	"github.com/Omotolani98/runner/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("monocron.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("|> Failed to connect to DB %v", err)
	}

	if err := DB.AutoMigrate(&models.EntryView{}); err != nil {
		return fmt.Errorf("migration failed: %v", err)
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}
