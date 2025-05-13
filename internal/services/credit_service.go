package services

import (
	"fmt"

	"social-credit/internal/models"

	"gorm.io/gorm"
)

type CreditService struct {
	db *gorm.DB
}

func NewCreditService(db *gorm.DB) *CreditService {
	return &CreditService{db: db}
}

func (s *CreditService) InitializeUser(userID int, username string, initialBalance int) error {
	user := models.Credit{UserID: userID, Username: username}
	result := s.db.FirstOrCreate(&user, models.Credit{UserID: userID})
	if result.RowsAffected > 0 {
		return s.db.Model(&user).Update("money", initialBalance).Error
	}
	return nil
}

func (s *CreditService) AddCredit(userID int, amount int) error {
	return s.db.Model(&models.Credit{}).
		Where("user_id = ?", userID).
		UpdateColumn("credit", gorm.Expr("credit + ?", amount)).
		Error
}

func (s *CreditService) GetUserCredit(userID int) (*models.Credit, error) {
	var credit models.Credit
	err := s.db.First(&credit, "user_id = ?", userID).Error
	return &credit, err
}

func (s *CreditService) GetTopCredits(limit int) ([]models.Credit, error) {
	var credits []models.Credit
	err := s.db.Order("credit DESC").Limit(limit).Find(&credits).Error
	return credits, err
}

func (s *CreditService) GetTopMoney(limit int) ([]models.Credit, error) {
	var credits []models.Credit
	err := s.db.Order("money DESC").Limit(limit).Find(&credits).Error
	return credits, err
}

func (s *CreditService) TransferMoney(senderID, receiverID int) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	var sender models.Credit
	if err := tx.First(&sender, "user_id = ?", senderID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if sender.Money <= 0 {
		tx.Rollback()
		return fmt.Errorf("insufficient balance")
	}

	if err := tx.Model(&models.Credit{}).
		Where("user_id = ?", senderID).
		UpdateColumn("money", gorm.Expr("money - ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&models.Credit{}).
		Where("user_id = ?", receiverID).
		UpdateColumn("money", gorm.Expr("money + ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *CreditService) UpdateUsername(userID int, newUsername string) error {
	return s.db.Model(&models.Credit{}).
		Where("user_id = ?", userID).
		Update("username", newUsername).Error
}
