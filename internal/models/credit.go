package models

type Credit struct {
	UserID     int `gorm:"primaryKey"`
	Username   string
	Credit     int
	Money      int `gorm:"default:0"`
	AliveScore int `gorm:"default:0"`
}
