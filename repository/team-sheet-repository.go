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
	Realm                   *string
	UniquesNeeded           *string
	Altars                  *string
	LookingForGroup         bool      `gorm:"not null;default:false"`
	UpdatedAt               time.Time `gorm:"not null;autoUpdateTime"`
}

type TeamSheetRepository interface {
	SaveEntry(entry *TeamSheetEntry) (*TeamSheetEntry, error)
	GetEntriesForEvent(eventId int) ([]*TeamSheetEntry, error)
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
	return entry, nil
}

func (r *TeamSheetRepositoryImpl) GetEntriesForEvent(eventId int) ([]*TeamSheetEntry, error) {
	entries := make([]*TeamSheetEntry, 0)
	result := r.DB.Preload("User.OauthAccounts").Find(&entries, TeamSheetEntry{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}
