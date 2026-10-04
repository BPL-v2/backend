package repository

import (
	"bpl/config"
	"bpl/metrics"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Signup struct {
	EventId          int       `gorm:"not null;references:event(id);primaryKey"`
	UserId           int       `gorm:"not null;references:event(id);primaryKey"`
	Timestamp        time.Time `gorm:"not null"`
	User             *User     `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
	ExpectedPlayTime int       `gorm:"not null"`
	NeedsHelp        bool      `gorm:"not null"`
	WantsToHelp      bool      `gorm:"not null"`
	Extra            *string   `gorm:"null"`
}

type SignupRepository interface {
	SaveSignup(signup *Signup) (*Signup, error)
	RemoveSignupForUser(userId int, eventId int) error
	GetSignupForUser(userId int, eventId int) (*Signup, error)
	GetSignupsForEvent(eventId int) ([]*Signup, error)
}

type SignupRepositoryImpl struct {
	DB *gorm.DB
}

func NewSignupRepository() SignupRepository {
	return &SignupRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *SignupRepositoryImpl) SaveSignup(signup *Signup) (*Signup, error) {
	result := r.DB.Save(signup)
	if result.Error != nil {
		return nil, result.Error
	}
	return signup, nil
}

func (r *SignupRepositoryImpl) RemoveSignupForUser(userId int, eventId int) error {
	result := r.DB.Delete(&Signup{}, &Signup{UserId: userId, EventId: eventId})
	return result.Error
}

func (r *SignupRepositoryImpl) GetSignupForUser(userId int, eventId int) (*Signup, error) {
	signup := Signup{}
	result := r.DB.First(&signup, &Signup{UserId: userId, EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return &signup, nil
}

func (r *SignupRepositoryImpl) GetSignupsForEvent(eventId int) ([]*Signup, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetSignupsForEvent"))
	defer timer.ObserveDuration()
	signups := make([]*Signup, 0)
	result := r.DB.Preload("User").Preload("User.OauthAccounts").Order("timestamp ASC").Find(&signups, &Signup{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return signups, nil
}
