package service

import (
	"bpl/client"
	"bpl/repository"
	"time"
)

type UniqueItemTrackingService interface {
	TrackUniqueItems(items []client.Item, teamId int, userId *int, eventId int, source repository.UniqueItemSource, timestamp time.Time) error
	GetByEventId(eventId int) ([]*repository.UniqueItemTracking, error)
	GetByTeamId(teamId int) ([]*repository.UniqueItemTracking, error)
}

type UniqueItemTrackingServiceImpl struct {
	trackingRepo repository.UniqueItemTrackingRepository
	itemService  ItemService
}

func NewUniqueItemTrackingService() UniqueItemTrackingService {
	return &UniqueItemTrackingServiceImpl{
		trackingRepo: repository.NewUniqueItemTrackingRepository(),
		itemService:  NewItemService(),
	}
}

func (s *UniqueItemTrackingServiceImpl) TrackUniqueItems(items []client.Item, teamId int, userId *int, eventId int, source repository.UniqueItemSource, timestamp time.Time) error {
	entries := make([]*repository.UniqueItemTracking, 0)
	for _, item := range items {
		if item.Rarity == nil || *item.Rarity != "Unique" {
			continue
		}
		itemRefId, err := s.itemService.GetOrCreateId(item.Name, repository.ItemTypeUnique)
		if err != nil {
			return err
		}
		entries = append(entries, &repository.UniqueItemTracking{
			ItemId:    *item.Id,
			ItemRefId: itemRefId,
			TeamId:    teamId,
			PlayerId:  userId,
			EventId:   eventId,
			Source:    source,
			Timestamp: timestamp,
		})
	}
	return s.trackingRepo.SaveBatch(entries)
}

func (s *UniqueItemTrackingServiceImpl) GetByEventId(eventId int) ([]*repository.UniqueItemTracking, error) {
	return s.trackingRepo.GetByEventId(eventId)
}

func (s *UniqueItemTrackingServiceImpl) GetByTeamId(teamId int) ([]*repository.UniqueItemTracking, error) {
	return s.trackingRepo.GetByTeamId(teamId)
}
