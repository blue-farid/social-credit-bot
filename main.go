package main

import (
	"fmt"
	"log"
	"net/http"
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
			Transfer []string `yaml:"transfer"`
		} `yaml:"stickers"`
		Capitalist struct {
			InitialBalance int `yaml:"initial_balance"`
		} `yaml:"capitalist"`
	} `yaml:"app"`
}

type Credit struct {
	UserID   int `gorm:"primaryKey"`
	Username string
	Credit   int
	Money    int `gorm:"default:0"`
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func main() {
	go func() {
		http.HandleFunc("/health", healthHandler)
		fmt.Println("Server started at :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()
	cfg, err := loadConfig()
	if err != nil {
		log.Panic("failed to load config: ", err)
	}
	bot, err := tgbotapi.NewBotAPI(os.Getenv("API_KEY"))
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

		if update.Message.From != nil {
			user := Credit{UserID: int(update.Message.From.ID), Username: update.Message.From.UserName}
			result := db.FirstOrCreate(&user, Credit{UserID: int(update.Message.From.ID)})
			if result.RowsAffected > 0 {
				db.Model(&user).Update("money", cfg.App.Capitalist.InitialBalance)
				msgText := fmt.Sprintf("💰 Welcome @%s! You received %d initial money.", user.Username, cfg.App.Capitalist.InitialBalance)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				bot.Send(msg)
			}
		}

		if update.Message.ReplyToMessage != nil && update.Message.Sticker != nil {
			if update.Message.From.ID == update.Message.ReplyToMessage.From.ID {
				cheater := Credit{UserID: int(update.Message.From.ID), Username: update.Message.From.UserName}
				db.FirstOrCreate(&cheater, Credit{UserID: int(update.Message.From.ID)})

				db.Model(&cheater).UpdateColumn("credit", gorm.Expr("credit - ?", 3))

				db.First(&cheater, "user_id = ?", update.Message.From.ID)

				msgText := fmt.Sprintf("🚫 Fraud detected! @%s tried to cheat by replying to their own message.\nPenalty: -3 SocialCredit\nCurrent balance: %d",
					cheater.Username,
					cheater.Credit)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				bot.Send(msg)
				continue
			}

			isTransferSticker := false
			for _, transferSticker := range cfg.App.Stickers.Transfer {
				if update.Message.Sticker.FileUniqueID == transferSticker {
					isTransferSticker = true
					break
				}
			}

			if isTransferSticker {
				if update.Message.From.ID == update.Message.ReplyToMessage.From.ID {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ You cannot transfer money to yourself!")
					bot.Send(msg)
					continue
				}

				sender := Credit{UserID: int(update.Message.From.ID)}
				receiver := Credit{UserID: int(update.Message.ReplyToMessage.From.ID)}

				db.First(&sender, "user_id = ?", sender.UserID)
				db.First(&receiver, "user_id = ?", receiver.UserID)

				if sender.Money <= 0 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ You don't have any money to transfer!")
					bot.Send(msg)
					continue
				}

				db.Model(&sender).UpdateColumn("money", gorm.Expr("money - ?", 1))
				db.Model(&receiver).UpdateColumn("money", gorm.Expr("money + ?", 1))

				db.First(&sender, "user_id = ?", sender.UserID)
				db.First(&receiver, "user_id = ?", receiver.UserID)

				msgText := fmt.Sprintf("💰 Money Transfer:\n@%s sent 1 money to @%s\n\n@%s's balance: %d\n@%s's balance: %d",
					sender.Username,
					receiver.Username,
					sender.Username,
					sender.Money,
					receiver.Username,
					receiver.Money)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				bot.Send(msg)
				continue
			}

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

		if update.Message.IsCommand() && update.Message.Command() == "money" {
			rows, err := db.Model(&Credit{}).
				Select("username, money").
				Order("money DESC").
				Limit(10).
				Rows()
			if err != nil {
				log.Println(err)
				continue
			}

			text := "💰 Money Leaderboard:\n"
			for rows.Next() {
				var username string
				var money int
				rows.Scan(&username, &money)
				text += fmt.Sprintf("@%s — %d\n", username, money)
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			bot.Send(msg)
		}
	}
}
