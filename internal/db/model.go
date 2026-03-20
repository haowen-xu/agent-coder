package db

import "time"

type SystemInfo struct {
	ID        uint   `gorm:"primaryKey"`
	Key       string `gorm:"size:128;uniqueIndex;not null"`
	Value     string `gorm:"size:1024;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
