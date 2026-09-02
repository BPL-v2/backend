package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type TeamSheetEntry struct {
	EventId                 int   `gorm:"not null;primaryKey"`
	UserId                  int   `gorm:"not null;primaryKey"`
	User                    *User `gorm:"foreignKey:UserId;references:Id;constraint:OnDelete:CASCADE"`
	CharacterName           *string
	Role                    *string
	Specialization          *string
	SecondaryRole           *string
	SecondarySpecialization *string
	Ascendancy              *string
	MainSkill               *string
	BuildNotes              *string
	PobUrl                  *string
	GuideUrl                *string
	Realm                   *string
	UniquesNeeded           *string
	Altars                  *string
	AltAscendancy           *string
	LookingForGroup         bool      `gorm:"not null;default:false"`
	UpdatedAt               time.Time `gorm:"not null;autoUpdateTime"`
}

type TeamSheetRepository interface {
	SaveEntry(entry *TeamSheetEntry) (*TeamSheetEntry, error)
	GetEntryForUser(eventId int, userId int) (*TeamSheetEntry, error)
	GetEntriesForUsers(eventId int, userIds []int) ([]*TeamSheetEntry, error)
}

type TeamSheetRepositoryImpl struct {
	DB *gorm.DB
}

func NewTeamSheetRepository() TeamSheetRepository {
	return &TeamSheetRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *TeamSheetRepositoryImpl) SaveEntry(entry *TeamSheetEntry) (*TeamSheetEntry, error) {
	if err := r.DB.Save(entry).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Preload("User.OauthAccounts").First(entry, TeamSheetEntry{EventId: entry.EventId, UserId: entry.UserId}).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *TeamSheetRepositoryImpl) GetEntryForUser(eventId int, userId int) (*TeamSheetEntry, error) {
	entry := &TeamSheetEntry{}
	result := r.DB.First(entry, TeamSheetEntry{EventId: eventId, UserId: userId})
	if result.Error != nil {
		return nil, result.Error
	}
	return entry, nil
}

func (r *TeamSheetRepositoryImpl) GetEntriesForUsers(eventId int, userIds []int) ([]*TeamSheetEntry, error) {
	entries := make([]*TeamSheetEntry, 0)
	result := r.DB.Preload("User.OauthAccounts").
		Where("event_id = ? AND user_id IN ?", eventId, userIds).
		Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}
