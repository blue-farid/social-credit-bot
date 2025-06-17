package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"

	"social-credit/internal/config"
	"social-credit/internal/models"
)

type ActivityService struct {
	bot           *tgbotapi.BotAPI
	config        *config.Config
	scheduler     *gocron.Scheduler
	db            *gorm.DB
	creditService *CreditService
}

func NewActivityService(bot *tgbotapi.BotAPI, config *config.Config, db *gorm.DB, creditService *CreditService) *ActivityService {
	return &ActivityService{
		bot:           bot,
		config:        config,
		scheduler:     gocron.NewScheduler(time.UTC),
		db:            db,
		creditService: creditService,
	}
}

func (s *ActivityService) Start() error {
	_, err := s.scheduler.Cron(s.config.App.ActivityCheck.Schedule).Do(s.checkAllUsersActivity)
	if err != nil {
		return fmt.Errorf("failed to schedule activity checks: %w", err)
	}
	s.scheduler.StartAsync()
	return nil
}

func (s *ActivityService) Stop() {
	s.scheduler.Stop()
}

func (s *ActivityService) checkAllUsersActivity() {
	var users []models.Credit
	if err := s.db.Find(&users).Error; err != nil {
		s.sendAlert("Error getting users for activity check: " + err.Error())
		return
	}

	for _, user := range users {
		s.checkUserActivity(&models.User{ID: int64(user.UserID), Username: user.Username})
	}
}

func (s *ActivityService) checkUserActivity(user *models.User) {
	var status models.ActivityStatus
	err := s.db.Where("user_id = ?", user.ID).First(&status).Error
	if err == gorm.ErrRecordNotFound {
		status = models.ActivityStatus{
			UserID:    user.ID,
			Username:  user.Username,
			IsActive:  true,
			LastCheck: time.Now(),
		}
	} else if err != nil {
		s.sendAlert(fmt.Sprintf("Error getting activity status for user %s: %s", user.Username, err.Error()))
		return
	}

	msg := tgbotapi.NewMessage(user.ID, "هی! زنده‌ای هنوز؟ لطفاً با دکمه زیر پاسخ بده.")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 بله، اینجام!", fmt.Sprintf("alive_%d", user.ID)),
		),
	)
	msg.ReplyMarkup = keyboard

	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		// who cares?
		return
	}

	status.MessageID = sentMsg.MessageID
	status.LastCheck = time.Now()
	status.NextCheckTime = time.Now().Add(time.Duration(s.config.App.ActivityCheck.ResponseTimeout) * time.Second)

	if err := s.db.Save(&status).Error; err != nil {
		s.sendAlert(fmt.Sprintf("Error saving activity status for user %s: %s", user.Username, err.Error()))
	}

	// Schedule check for response timeout
	s.scheduler.Every(s.config.App.ActivityCheck.ResponseTimeout).Seconds().Do(func() {
		s.checkResponseTimeout(&status)
	})
}

func (s *ActivityService) checkResponseTimeout(status *models.ActivityStatus) {
	if time.Now().Before(status.NextCheckTime) {
		return
	}

	if !status.IsActive {
		return
	}

	status.RetryCount++
	if status.RetryCount >= s.config.App.ActivityCheck.MaxRetries {
		status.IsActive = false
		s.sendAlert(fmt.Sprintf("کاربر %s دیگه جواب نمیده! غیرفعال شد. 💀", status.Username))
	} else {
		s.sendWarning(fmt.Sprintf("کاربر %s هنوز جواب نداده! %d بار دیگه چک می‌کنیم.", status.Username, s.config.App.ActivityCheck.MaxRetries-status.RetryCount))
		status.NextCheckTime = time.Now().Add(time.Duration(s.config.App.ActivityCheck.RetryInterval) * time.Second)
		s.checkUserActivity(&models.User{ID: status.UserID, Username: status.Username})
	}

	if err := s.db.Save(&status).Error; err != nil {
		s.sendAlert(fmt.Sprintf("Error saving activity status for user %s: %s", status.Username, err.Error()))
	}
}

func (s *ActivityService) HandleAliveResponse(userID int64, username string) {
	var status models.ActivityStatus
	err := s.db.Where("user_id = ?", userID).First(&status).Error
	if err != nil {
		s.sendAlert(fmt.Sprintf("Error getting activity status for user %s: %s", username, err.Error()))
		return
	}

	if !status.IsActive {
		return
	}

	status.LastResponse = time.Now()
	status.RetryCount = 0
	status.IsActive = true

	if err := s.db.Save(&status).Error; err != nil {
		s.sendAlert(fmt.Sprintf("Error saving activity status for user %s: %s", username, err.Error()))
		return
	}

	// Save activity check record
	check := &models.ActivityCheck{
		UserID:    userID,
		Username:  username,
		CheckTime: time.Now(),
		Response:  true,
		Score:     1,
	}
	if err := s.db.Create(check).Error; err != nil {
		s.sendAlert(fmt.Sprintf("Error saving activity check for user %s: %s", username, err.Error()))
		return
	}

	// Award points for being alive
	err = s.creditService.AwardPoints(context.Background(), userID, s.config.App.ActivityCheck.Rewards.AliveScore, "زنده موندن")
	if err != nil {
		s.sendAlert(fmt.Sprintf("Error awarding points to user %s: %s", username, err.Error()))
		return
	}

	s.sendAlert(fmt.Sprintf("کاربر %s زنده است! 🎉 %d امتیاز دریافت کرد.", username, s.config.App.ActivityCheck.Rewards.AliveScore))
}

func (s *ActivityService) sendAlert(message string) {
	chatID, err := strconv.ParseInt(s.config.App.ActivityCheck.Channels.Alerts, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing alert channel ID: %s\n", err.Error())
		return
	}
	msg := tgbotapi.NewMessage(chatID, message)
	_, err = s.bot.Send(msg)
	if err != nil {
		fmt.Printf("Error sending alert: %s\n", err.Error())
	}
}

func (s *ActivityService) sendWarning(message string) {
	chatID, err := strconv.ParseInt(s.config.App.ActivityCheck.Channels.Warnings, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing warning channel ID: %s\n", err.Error())
		return
	}
	msg := tgbotapi.NewMessage(chatID, message)
	_, err = s.bot.Send(msg)
	if err != nil {
		fmt.Printf("Error sending warning: %s\n", err.Error())
	}
}
