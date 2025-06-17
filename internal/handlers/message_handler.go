package handlers

import (
	"fmt"
	"log"

	"social-credit/internal/config"
	"social-credit/internal/services"

	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageHandler struct {
	bot             *tgbotapi.BotAPI
	config          *config.Config
	credit          *services.CreditService
	activityService *services.ActivityService
}

func NewMessageHandler(bot *tgbotapi.BotAPI, cfg *config.Config, credit *services.CreditService, activityService *services.ActivityService) *MessageHandler {
	return &MessageHandler{
		bot:             bot,
		config:          cfg,
		credit:          credit,
		activityService: activityService,
	}
}

func (h *MessageHandler) HandleMessage(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Data[:6] == "alive_" {
			userID := update.CallbackQuery.From.ID
			username := update.CallbackQuery.From.UserName
			h.activityService.HandleAliveResponse(userID, username)

			// Get user's credit info to show their alive score
			userCredit, err := h.credit.GetUserCredit(int(userID))
			if err != nil {
				log.Printf("Error getting user credit: %v", err)
				return
			}

			// Send response message
			responseText := fmt.Sprintf("✅ حضور شما ثبت شد!\nامتیاز زنده بودن شما: %d", userCredit.AliveScore)
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, responseText)
			h.bot.Send(msg)

			// Remove the "loading" state from the button
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			h.bot.Request(callback)

			// Edit the original message to show it's been answered
			editMsg := tgbotapi.NewEditMessageText(
				update.CallbackQuery.Message.Chat.ID,
				update.CallbackQuery.Message.MessageID,
				"هی! زنده‌ای هنوز؟ ✅ بله!",
			)
			h.bot.Send(editMsg)
			return
		}
	}

	if update.Message == nil {
		return
	}

	if update.Message.From != nil {
		existingUser, err := h.credit.GetUserCredit(int(update.Message.From.ID))
		if err != nil {
			err := h.credit.InitializeUser(
				int(update.Message.From.ID),
				update.Message.From.UserName,
				h.config.App.Capitalist.InitialBalance,
			)
			if err == nil {
				msgText := fmt.Sprintf("💰 Welcome @%s! You received %d initial money.",
					update.Message.From.UserName,
					h.config.App.Capitalist.InitialBalance)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				h.bot.Send(msg)
			}
		} else {
			if existingUser.Username != update.Message.From.UserName {
				h.credit.UpdateUsername(int(update.Message.From.ID), update.Message.From.UserName)
			}
		}
	}

	if update.Message.ReplyToMessage != nil && update.Message.Sticker != nil {
		h.handleStickerReply(update)
		return
	}

	if update.Message.IsCommand() {
		h.handleCommand(update)
	}
}

func (h *MessageHandler) handleStickerReply(update tgbotapi.Update) {
	if update.Message.From.ID == update.Message.ReplyToMessage.From.ID {
		if h.handleSelfReplyFraud(update) {
			return
		}
	}

	if h.isTransferSticker(update.Message.Sticker.FileUniqueID) {
		h.handleMoneyTransfer(update)
		return
	}

	h.handleSocialCredit(update)
}

func (h *MessageHandler) handleSelfReplyFraud(update tgbotapi.Update) bool {
	if h.getStickerType(update.Message.Sticker.FileUniqueID) != "positive" {
		return false
	}

	cheater, err := h.credit.GetUserCredit(int(update.Message.From.ID))
	if err != nil {
		log.Printf("Error getting user credit: %v", err)
		return false
	}

	h.credit.AddCredit(int(update.Message.From.ID), -3)
	msgText := fmt.Sprintf("🚫 Fraud detected! @%s tried to cheat by replying to their own message with a positive sticker.\nPenalty: -3 SocialCredit\nCurrent balance: %d",
		cheater.Username,
		cheater.Credit-3)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	h.bot.Send(msg)
	return true
}

func (h *MessageHandler) isTransferSticker(fileUniqueID string) bool {
	return slices.Contains(h.config.App.Stickers.Transfer, fileUniqueID)
}

func (h *MessageHandler) handleMoneyTransfer(update tgbotapi.Update) {
	err := h.credit.TransferMoney(
		int(update.Message.From.ID),
		int(update.Message.ReplyToMessage.From.ID),
	)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ You don't have any money to transfer!")
		h.bot.Send(msg)
		return
	}

	sender, _ := h.credit.GetUserCredit(int(update.Message.From.ID))
	receiver, _ := h.credit.GetUserCredit(int(update.Message.ReplyToMessage.From.ID))

	msgText := fmt.Sprintf("💰 Money Transfer:\n@%s sent 1 money to @%s\n\n@%s's balance: %d\n@%s's balance: %d",
		sender.Username,
		receiver.Username,
		sender.Username,
		sender.Money,
		receiver.Username,
		receiver.Money)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	h.bot.Send(msg)
}

func (h *MessageHandler) handleSocialCredit(update tgbotapi.Update) {
	stickerType := h.getStickerType(update.Message.Sticker.FileUniqueID)
	if stickerType == "" {
		return
	}

	amount := 1
	if stickerType == "negative" {
		amount = -1
	}

	user, err := h.credit.GetUserCredit(int(update.Message.ReplyToMessage.From.ID))
	if err != nil {
		log.Printf("Error getting user credit: %v", err)
		return
	}

	h.credit.AddCredit(int(update.Message.ReplyToMessage.From.ID), amount)
	msgText := fmt.Sprintf("@%s got %+d SocialCredit! Total: %d",
		user.Username,
		amount,
		user.Credit+amount)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	h.bot.Send(msg)
}

func (h *MessageHandler) getStickerType(fileUniqueID string) string {
	if slices.Contains(h.config.App.Stickers.Positive, fileUniqueID) {
		return "positive"
	}
	if slices.Contains(h.config.App.Stickers.Negative, fileUniqueID) {
		return "negative"
	}
	return ""
}

func (h *MessageHandler) handleCommand(update tgbotapi.Update) {
	switch update.Message.Command() {
	case "credits":
		h.handleCreditsCommand(update)
	case "money":
		h.handleMoneyCommand(update)
	case "alive":
		h.handleAliveCommand(update)
	}
}

func (h *MessageHandler) handleCreditsCommand(update tgbotapi.Update) {
	credits, err := h.credit.GetTopCredits(10)
	if err != nil {
		log.Printf("Error getting top credits: %v", err)
		return
	}

	text := "🌟 SocialCredit Leaderboard:\n"
	for _, credit := range credits {
		text += fmt.Sprintf("@%s — %d\n", credit.Username, credit.Credit)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	h.bot.Send(msg)
}

func (h *MessageHandler) handleMoneyCommand(update tgbotapi.Update) {
	credits, err := h.credit.GetTopMoney(10)
	if err != nil {
		log.Printf("Error getting top money: %v", err)
		return
	}

	text := "💰 Money Leaderboard:\n"
	for _, credit := range credits {
		text += fmt.Sprintf("@%s — %d\n", credit.Username, credit.Money)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	h.bot.Send(msg)
}

func (h *MessageHandler) handleAliveCommand(update tgbotapi.Update) {
	credits, err := h.credit.GetTopAliveScores(10)
	if err != nil {
		log.Printf("Error getting top alive scores: %v", err)
		return
	}

	text := "🌟 امتیاز زنده بودن:\n"
	for _, credit := range credits {
		text += fmt.Sprintf("@%s — %d\n", credit.Username, credit.AliveScore)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	h.bot.Send(msg)
}
