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
	AutoCheckKey *AchievementCheckKey `gorm:"column:auto_check_key"`
	EventId      *int                 `gorm:"column:event_id"`
}

type AchievementCheckKey string

const (
	CheckLevel90              AchievementCheckKey = "level_90"
	CheckLevel95              AchievementCheckKey = "level_95"
	CheckLevel100             AchievementCheckKey = "level_100"
	CheckParticipatedInEvent  AchievementCheckKey = "participated_in_event"
	CheckPlayed5Leagues       AchievementCheckKey = "played_5_leagues"
	CheckPlayed10Leagues      AchievementCheckKey = "played_10_leagues"
	CheckPlayed5Ascendancies  AchievementCheckKey = "played_5_ascendancies"
	CheckPlayed10Ascendancies AchievementCheckKey = "played_10_ascendancies"
	CheckTeamlead             AchievementCheckKey = "teamlead"
	CheckSubmittedBounty      AchievementCheckKey = "submitted_bounty"
)

type UserAchievement struct {
	UserId        int       `gorm:"primaryKey"`
	AchievementId int       `gorm:"primaryKey"`
	GrantedAt     time.Time `gorm:"not null;default:now()"`
	GrantedBy     *int

	User *User `gorm:"foreignKey:UserId"`
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
