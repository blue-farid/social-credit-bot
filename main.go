package main

import (
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	App struct {
		Test     bool `yaml:"test"`
		Database struct {
			Type  string `yaml:"type"`
			MySQL struct {
				Host     string `yaml:"host"`
				Port     int    `yaml:"port"`
				User     string `yaml:"user"`
				Password string `yaml:"password"`
				DBName   string `yaml:"dbname"`
			} `yaml:"mysql"`
			SQLite struct {
				Path string `yaml:"path"`
			} `yaml:"sqlite"`
		} `yaml:"database"`
		Stickers struct {
			Positive []string `yaml:"positive"`
			Negative []string `yaml:"negative"`
		} `yaml:"stickers"`
	} `yaml:"app"`
}

type Credit struct {
	UserID   int `gorm:"primaryKey"`
	Username string
	Credit   int
}

func loadConfig() (*Config, error) {
	f, err := os.Open("config.yaml")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	return &cfg, err
}

func main() {
	cfg, err := loadConfig()
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

	db.AutoMigrate(&Credit{})

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.ReplyToMessage != nil && update.Message.Sticker != nil {
			stickerType := ""
			for _, positiveSticker := range cfg.App.Stickers.Positive {
				if update.Message.Sticker.FileUniqueID == positiveSticker {
					stickerType = "positive"
					break
				}
			}
			if stickerType == "" {
				for _, negativeSticker := range cfg.App.Stickers.Negative {
					if update.Message.Sticker.FileUniqueID == negativeSticker {
						stickerType = "negative"
						break
					}
				}
			}

			if stickerType == "" {
				continue
			}

			uid := update.Message.ReplyToMessage.From.ID
			user := update.Message.ReplyToMessage.From.UserName

			c := Credit{UserID: int(uid), Username: user}
			db.FirstOrCreate(&c, Credit{UserID: int(uid)})

			if stickerType == "positive" {
				db.Model(&c).UpdateColumn("credit", gorm.Expr("credit + ?", 1))
				msgText := fmt.Sprintf("@%s got +1 SocialCredit! Total: %d", c.Username, c.Credit+1)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				bot.Send(msg)
			} else {
				db.Model(&c).UpdateColumn("credit", gorm.Expr("credit - ?", 1))
				msgText := fmt.Sprintf("@%s got -1 SocialCredit! Total: %d", c.Username, c.Credit-1)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				bot.Send(msg)
			}
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "credits" {
			rows, err := db.Model(&Credit{}).
				Select("username, credit").
				Order("credit DESC").
				Limit(10).
				Rows()
			if err != nil {
				log.Println(err)
				continue
			}

			text := "🌟 SocialCredit Leaderboard:\n"
			for rows.Next() {
				var username string
				var points int
				rows.Scan(&username, &points)
				text += fmt.Sprintf("@%s — %d\n", username, points)
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			bot.Send(msg)
		}
	}
}
