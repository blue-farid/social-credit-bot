package main

import (
	"fmt"
	"log"
	"net/http"

	"social-credit/internal/config"
	"social-credit/internal/handlers"
	"social-credit/internal/models"
	"social-credit/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Panic("failed to load config: ", err)
	}

	healthHandler := handlers.NewHealthHandler()
	go func() {
		log.Printf("Starting health check server on port 8080")
		if err := http.ListenAndServe(":8080", healthHandler); err != nil {
			log.Printf("Health check server error: %v", err)
		}
	}()

	bot, err := tgbotapi.NewBotAPI(cfg.App.Token)
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
	} else if cfg.App.Database.Type == "postgres" {
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
			cfg.App.Database.Postgres.Host,
			cfg.App.Database.Postgres.Port,
			cfg.App.Database.Postgres.User,
			cfg.App.Database.Postgres.Password,
			cfg.App.Database.Postgres.DBName)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Panic("failed to connect to PostgreSQL database: ", err)
		}
		log.Println("Connected to PostgreSQL database")
	} else {
		log.Panic("unsupported database type: ", cfg.App.Database.Type)
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
