package models

import (
	"time"
)

// ActivityStatus represents the current status of a user's activity check
type ActivityStatus struct {
	UserID        int64     `gorm:"primaryKey"`
	Username      string    `gorm:"not null"`
	LastCheck     time.Time `gorm:"not null"`
	LastResponse  time.Time
	RetryCount    int  `gorm:"not null;default:0"`
	IsActive      bool `gorm:"not null;default:true"`
	MessageID     int
	NextCheckTime time.Time `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// ActivityCheck represents a single activity check instance
type ActivityCheck struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"not null;index"`
	Username  string    `gorm:"not null"`
	CheckTime time.Time `gorm:"not null;index"`
	Response  bool      `gorm:"not null"`
	Score     int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
