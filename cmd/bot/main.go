package main

import (
	"fmt"
	"log"

	"social-credit/internal/config"
	"social-credit/internal/handlers"
	"social-credit/internal/models"
	"social-credit/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Panic("failed to load config: ", err)
	}

	bot, err := tgbotapi.NewBotAPI("7780232983:AAHc_AActaCvmBr40oG_y29JGKZe_aYZrfE")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	var db *gorm.DB
	if cfg.App.Test || cfg.App.Database.Type == "sqlite" {
		db, err = gorm.Open(sqlite.Open(cfg.App.Database.SQLite.Path), &gorm.Config{})
		if err != nil {
			log.Panic("failed to connect to SQLite database: ", err)
		}
		log.Println("Connected to SQLite database in test mode")
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.App.Database.MySQL.User,
			cfg.App.Database.MySQL.Password,
			cfg.App.Database.MySQL.Host,
			cfg.App.Database.MySQL.Port,
			cfg.App.Database.MySQL.DBName)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Panic("failed to connect to MySQL database: ", err)
		}
		log.Println("Connected to MySQL database")
	}

	db.AutoMigrate(&models.Credit{})

	creditService := services.NewCreditService(db)

	messageHandler := handlers.NewMessageHandler(bot, cfg, creditService)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		messageHandler.HandleMessage(update)
	}
}
