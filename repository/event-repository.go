package repository

import (
	"bpl/client"
	"bpl/config"
	"bpl/utils"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
)

type GameVersion string

const (
	PoE1 GameVersion = "poe1"
	PoE2 GameVersion = "poe2"
)

type Event struct {
	Id                   int          `gorm:"primaryKey"`
	Name                 string       `gorm:"not null"`
	IsCurrent            bool         `gorm:"not null"`
	GameVersion          GameVersion  `gorm:"not null"`
	Patch                *string      `gorm:"null"`
	MaxSize              int          `gorm:"not null"`
	WaitlistSize         int          `gorm:"not null"`
	ApplicationStartTime time.Time    `gorm:"not null"`
	ApplicationEndTime   time.Time    `gorm:"not null"`
	EventStartTime       time.Time    `gorm:"not null"`
	EventEndTime         time.Time    `gorm:"not null"`
	Public               bool         `gorm:"not null"`
	Locked               bool         `gorm:"not null"`
	IsMainEvent          bool         `gorm:"not null"`
	UsesMedals           bool         `gorm:"not null"`
	Teams                []*Team      `gorm:"foreignKey:EventId;constraint:OnDelete:CASCADE"`
	Objectives           []*Objective `gorm:"foreignKey:EventId;constraint:OnDelete:CASCADE"`
}

func (e *Event) GetRealm() *client.Realm {
	if e.GameVersion == PoE2 {
		realm := client.PoE2
		return &realm
	}
	return nil
}

func (e *Event) TeamIds() []int {
	return utils.Map(e.Teams, func(t *Team) int { return t.Id })
}

type EventRepository interface {
	GetCurrentEvent(preloads ...string) (*Event, error)
	GetEventById(eventId int, preloads ...string) (*Event, error)
	InvalidateCurrentEvent() error
	Delete(event *Event) error
	FindAll(preloads ...string) ([]*Event, error)
	SaveEvent(event *Event) (*Event, error)
	GetEventByConditionId(conditionId int) (*Event, error)
}

type EventRepositoryImpl struct {
	DB *gorm.DB
}

func NewEventRepository() EventRepository {
	return &EventRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *EventRepositoryImpl) GetCurrentEvent(preloads ...string) (*Event, error) {
	var event *Event
	query := r.DB

	for _, preload := range preloads {
		if preload == "Teams.Users" {
			continue
		}
		query = query.Preload(preload)
	}

	result := query.Where(Event{IsCurrent: true}).First(&event)
	if result.Error != nil {
		return nil, fmt.Errorf("no current event found: %v", result.Error)
	}
	if len(preloads) > 0 && slices.Contains(preloads, "Teams.Users") {
		err := LoadUsersIntoEvent(r.DB, event)
		if err != nil {
			return nil, fmt.Errorf("failed to load users into event: %v", err)
		}
	}

	return event, nil
}

func (r *EventRepositoryImpl) GetEventById(eventId int, preloads ...string) (*Event, error) {
	var event *Event
	query := r.DB

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	result := query.First(&event, eventId)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find event: %v", result.Error)
	}
	if len(preloads) > 0 && slices.Contains(preloads, "Teams.Users") {
		err := LoadUsersIntoEvent(r.DB, event)
		if err != nil {
			return nil, fmt.Errorf("failed to load users into event: %v", err)
		}
	}
	return event, nil
}

func (r *EventRepositoryImpl) InvalidateCurrentEvent() error {
	result := r.DB.Exec("UPDATE events SET is_current = false WHERE is_current = true")
	if result.Error != nil {
		return fmt.Errorf("failed to invalidate current event: %v", result.Error)
	}
	return nil
}

func (r *EventRepositoryImpl) Delete(event *Event) error {
	return r.DB.Delete(&event).Error
}

func (r *EventRepositoryImpl) FindAll(preloads ...string) ([]*Event, error) {
	var events []*Event
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&events)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find events: %v", result.Error)
	}
	return events, nil
}

func (r *EventRepositoryImpl) SaveEvent(event *Event) (*Event, error) {
	result := r.DB.Save(event)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save event: %v", result.Error)
	}
	return event, nil
}

func (r *EventRepositoryImpl) GetEventByConditionId(conditionId int) (*Event, error) {
	var event Event
	query := `
		SELECT * FROM events
		JOIN objectives ON objectives.event_id = events.id
		JOIN conditions ON objectives.id = conditions.objective_id
		WHERE conditions.id = ?
	`
	result := r.DB.Raw(query, conditionId).Scan(&event)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find event by condition id: %v", result.Error)
	}
	return &event, nil
}
