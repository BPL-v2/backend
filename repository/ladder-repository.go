package repository

import (
	"bpl/client"
	"bpl/config"
	"bpl/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type LadderEntry struct {
	UserId        *int    `gorm:"index"`
	Account       string  `gorm:"not null"`
	Character     string  `gorm:"not null"`
	Class         string  `gorm:"not null"`
	Level         int     `gorm:"not null"`
	Delve         int     `gorm:"not null"`
	Experience    int     `gorm:"not null"`
	Rank          int     `gorm:"not null"`
	TwitchAccount *string `gorm:"null"`
	EventId       int     `gorm:"foreignKey:EventId;constraint:OnDelete:CASCADE;index;not null"`
}

type LadderRepository interface {
	UpsertLadder(ladder []*client.LadderEntry, eventId int, playerMap map[string]int) error
	GetLadderForEvent(eventId int) ([]*LadderEntry, error)
}

type LadderRepositoryImpl struct {
	DB *gorm.DB
}

func NewLadderRepository() LadderRepository {
	return &LadderRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *LadderRepositoryImpl) UpsertLadder(ladder []*client.LadderEntry, eventId int, playerMap map[string]int) error {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("UpsertLadder"))
	defer timer.ObserveDuration()

	err := r.DB.Delete(&LadderEntry{}, &LadderEntry{EventId: eventId}).Error
	if err != nil {
		return err
	}
	dbEntries := make([]*LadderEntry, 0, len(ladder))
	for _, entry := range ladder {
		dbEntry := &LadderEntry{
			Character: entry.Character.Name,
			Level:     entry.Character.Level,
			Class:     entry.Character.Class,
			EventId:   eventId,
			Rank:      entry.Rank,
		}
		if entry.Account != nil {
			dbEntry.Account = entry.Account.Name
		}
		if userId, exists := playerMap[entry.Character.Name]; exists {
			dbEntry.UserId = &userId
		}
		if entry.Character.Depth != nil && entry.Character.Depth.Default != nil {
			dbEntry.Delve = *entry.Character.Depth.Default
		}
		if entry.Character.Experience != nil {
			dbEntry.Experience = *entry.Character.Experience
		}
		if entry.Account != nil && entry.Account.Twitch != nil {
			dbEntry.TwitchAccount = &entry.Account.Twitch.Name
		}
		dbEntries = append(dbEntries, dbEntry)
	}
	return r.DB.CreateInBatches(dbEntries, 500).Error
}

func (r *LadderRepositoryImpl) GetLadderForEvent(eventId int) ([]*LadderEntry, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetLadderForEvent"))
	defer timer.ObserveDuration()
	var ladder []*LadderEntry
	result := r.DB.Find(&ladder, &LadderEntry{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return ladder, nil
}
