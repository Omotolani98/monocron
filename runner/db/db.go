package db

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

func InitDB() (*DB, error) {
	db, err := gorm.Open(sqlite.Open("monocron.db"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("|> Failed to connect to DB %v", err)
	}

	return &DB{
		db: db,
	}, nil
}
