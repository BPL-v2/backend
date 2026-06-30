package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type Achievement struct {
	Id           int    `gorm:"primaryKey;autoIncrement"`
	Name         string `gorm:"unique;not null"`
	Description  string `gorm:"not null;default:''"`
	IsCustom     bool   `gorm:"not null;default:false"`
	Icon         []byte `gorm:"type:bytea"`
	IconMimeType string
}

type UserAchievement struct {
	UserId        int       `gorm:"primaryKey"`
	AchievementId int       `gorm:"primaryKey"`
	GrantedAt     time.Time `gorm:"not null;default:now()"`
	GrantedBy     *int

	User *User `gorm:"foreignKey:UserId"`
}

const (
	AchievementReachedLvl90                 = "Reached level 90"
	AchievementReachedLvl95                 = "Reached level 95"
	AchievementReachedLvl100                = "Reached level 100"
	AchievementParticipated                 = "Participated in an event"
	AchievementPlayed5Leagues               = "Played 5 leagues"
	AchievementPlayed10Leagues              = "Played 10 leagues"
	AchievementPlayed10DifferentAscendancies = "Played 10 different ascendancies"
)

// SystemAchievements lists all achievement names that are auto-awarded by the sync job.
var SystemAchievements = []string{
	"Participated in an event",
	"Won an event",
	"Teamlead",
	"Played 5 leagues",
	"Played 10 leagues",
	"Reached level 90",
	"Reached level 95",
	"Reached level 100",
	"Submitted a bounty",
	"Submitted a point unique",
	"Played 5 different ascendancies",
	"Played 10 different ascendancies",
}

type AchievementRepository interface {
	GetAllAchievements() ([]*Achievement, error)
	GetAchievementById(id int) (*Achievement, error)
	SaveAchievement(achievement *Achievement) (*Achievement, error)
	SaveIcon(id int, icon []byte, mimeType string) error
	DeleteAchievement(id int) error

	GetAllUserAchievements(userId *int) ([]*UserAchievement, error)
	SaveUserAchievement(ua *UserAchievement) error
	SaveUserAchievements(uas []*UserAchievement) error
	DeleteUserAchievement(userId, achievementId int) error
}

type AchievementRepositoryImpl struct {
	DB *gorm.DB
}

func NewAchievementRepository() AchievementRepository {
	return &AchievementRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *AchievementRepositoryImpl) GetAllAchievements() ([]*Achievement, error) {
	var achievements []*Achievement
	err := r.DB.Find(&achievements).Error
	return achievements, err
}

func (r *AchievementRepositoryImpl) GetAchievementById(id int) (*Achievement, error) {
	var achievement Achievement
	err := r.DB.First(&achievement, id).Error
	return &achievement, err
}

func (r *AchievementRepositoryImpl) SaveAchievement(achievement *Achievement) (*Achievement, error) {
	err := r.DB.Save(achievement).Error
	return achievement, err
}

func (r *AchievementRepositoryImpl) SaveIcon(id int, icon []byte, mimeType string) error {
	return r.DB.Model(&Achievement{}).Where("id = ?", id).Updates(map[string]any{
		"icon":           icon,
		"icon_mime_type": mimeType,
	}).Error
}

func (r *AchievementRepositoryImpl) DeleteAchievement(id int) error {
	return r.DB.Delete(&Achievement{}, id).Error
}

func (r *AchievementRepositoryImpl) GetAllUserAchievements(userId *int) ([]*UserAchievement, error) {
	var uas []*UserAchievement
	q := r.DB
	if userId != nil {
		q = q.Where("user_id = ?", *userId)
	}
	err := q.Find(&uas).Error
	return uas, err
}

func (r *AchievementRepositoryImpl) SaveUserAchievement(ua *UserAchievement) error {
	return r.DB.Save(ua).Error
}

func (r *AchievementRepositoryImpl) SaveUserAchievements(uas []*UserAchievement) error {
	return r.DB.Save(&uas).Error
}

func (r *AchievementRepositoryImpl) DeleteUserAchievement(userId, achievementId int) error {
	return r.DB.Delete(&UserAchievement{}, "user_id = ? AND achievement_id = ?", userId, achievementId).Error
}
